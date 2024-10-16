package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/go-extract"
	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
	"github.com/hashicorp/go-slug"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// ExtractorFkt is a function pointer to implement the extraction function
type ExtractorFkt func(context.Context, io.Reader, string) error

var CLI struct {
	CacheInMemory bool     `short:"c" long:"cache-in-memory" default:"false" description:"cache input files in memory"`
	Extract       bool     `short:"e" long:"extract" default:"false" description:"use the go-extract extraction method"`
	InputArchives []string `arg:"" name:"input-archives" required:"true" description:"input archives to extract"`
	Iterations    int      `short:"i" long:"iterations" default:"1" description:"number of iterations to repeat the extraction"`
	Profile       bool     `short:"p" long:"profile" default:"false" description:"enable profiling of the extraction"`
	ProfileOut    string   `short:"o" long:"profile-out" default:"mem.pprof" description:"output file for the profile"`
	Parallel      bool     `short:"P" long:"parallel" default:"false" description:"use both methods in parallel"`
	SrcFromMem    bool     `short:"m" long:"src-from-mem" default:"false" description:"read input files into memory"`
	Slug          bool     `short:"s" long:"slug" default:"false" description:"use the slug extraction method"`
	Verbose       bool     `short:"v" long:"verbose" description:"Enable verbose output"`
}

// main function
func main() {

	// parse command line arguments and create logger
	var ctx = context.Background()
	_ = kong.Parse(&CLI)
	lvl := slog.LevelInfo
	if CLI.Verbose {
		lvl = slog.LevelDebug
	}
	var logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))

	// declare extraction functions
	extractorFunctions := map[string]ExtractorFkt{}

	// check if go-extract implementation is requested
	if CLI.Extract {
		extractorFunctions["go-extract"] = extractWithGoExtract
	}

	// check if go-slug implementation is requested
	if CLI.Slug {
		extractorFunctions["go-slug"] = extractWithSlug
	}

	// check if parallel implementation is requested
	if CLI.Parallel {
		extractorFunctions["parallel"] = extractParallel
	}

	if len(extractorFunctions) == 0 {
		logger.Error("no extraction method specified, using go-extract as default")
		extractorFunctions["go-extract"] = extractWithGoExtract
	}
	// map with slice of int to capture execution duration
	var ed = make(map[string][]int64)

	// repeat 10 times
	for i := 0; i < CLI.Iterations; i++ {
		// iterate over filenames
		for _, filename := range CLI.InputArchives {
			// iterate over extraction functions
			for extractionMethod, extractionFunction := range extractorFunctions {
				if duration, err := profileExtraction(ctx, logger, filename, extractionMethod, extractionFunction); err != nil {
					logger.Error("error during extraction", "error", err)
				} else {
					key := fmt.Sprintf("%s-%s", filename, extractionMethod)
					ed[key] = append(ed[key], duration)
				}
			}
		}
	}

	// log average, min and max duration
	for _, key := range sortedKeys(ed) {
		logger.Info("extraction profiling results", "iterations", len(ed[key]), "average", fmt.Sprintf("%dms", avg(ed[key])), "min", fmt.Sprintf("%dms", min(ed[key])), "max", fmt.Sprintf("%dms", max(ed[key])), "std", fmt.Sprintf("%dms", int(std(ed[key]))), "key", key)
	}

	// store memory profile
	if CLI.Profile {
		// write memory profile
		logger.Debug("writing memory profile", "filename", CLI.ProfileOut)
		logger.Info(fmt.Sprintf("analyze with: go tool pprof -http=:8080 %s", CLI.ProfileOut))
		f, err := os.Create(CLI.ProfileOut)
		if err != nil {
			logger.Error("error creating memory profile", "error", err)
		}
		defer f.Close()
		err = pprof.WriteHeapProfile(f)
		if err != nil {
			logger.Error("error writing memory profile", "error", err)
		}
	}

}

var td []telemetry.Data

func storeTelemetryData(ctx context.Context, d *telemetry.Data) {
	td = append(td, *d)
}

// extractWithSlug extracts the given reader to the given target
func extractWithSlug(ctx context.Context, reader io.Reader, tmpExtractTarget string) error {
	return slug.Unpack(reader, tmpExtractTarget)
}

// unpackConfigGoExtract is a config for go-extract that matches the
// behavior of the previous slug.Unpack. This is used to ensure that
// the new implementation is a drop-in replacement for the old one.
var unpackConfigGoExtract = config.NewConfig(
	config.WithContinueOnError(true),             // taken from go-slug
	config.WithContinueOnUnsupportedFiles(true),  // taken from go-slug
	config.WithDenySymlinkExtraction(true),       // taken from go-slug
	config.WithMaxExtractionSize(-1),             // disable check for now
	config.WithMaxFiles(-1),                      // disable check for now
	config.WithMaxInputSize(-1),                  // disable check for now
	config.WithTelemetryHook(storeTelemetryData), // store telemetry data
)

// extractWithGoExtract extracts the given reader to the given target
func extractWithGoExtract(ctx context.Context, reader io.Reader, tmpExtractTarget string) error {
	return extract.Unpack(ctx, reader, tmpExtractTarget, unpackConfigGoExtract)
}

// extractParallel extracts the provided input with both approaches
func extractParallel(ctx context.Context, reader io.Reader, tmpExtractTarget string) error {
	return unpackParallel(ctx, tmpExtractTarget, reader)
}

// sortedKeys returns the keys of the given map in sorted order
func sortedKeys(m map[string][]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return sort.StringSlice(keys)
}

type noSeeker struct {
	r io.Reader
}

func (n *noSeeker) Read(p []byte) (int, error) {
	return n.r.Read(p)
}

// profileExtraction extracts the profile from the given file and calls the given extraction function
func profileExtraction(ctx context.Context, logger *slog.Logger, filename string, libraryName string, extractionFunction ExtractorFkt) (int64, error) {

	// create temporary directory
	tmpExtract, err := os.MkdirTemp("", "extract-*")
	if err != nil {
		return -1, fmt.Errorf("error creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpExtract)

	// open file
	inf, err := os.Open(filename)
	if err != nil {
		return -1, fmt.Errorf("error opening file: %w", err)
	}
	defer inf.Close()
	reader := io.Reader(inf)

	// read file into memory
	if CLI.SrcFromMem {
		// read file into memory
		b, err := os.ReadFile(filename)
		if err != nil {
			return -1, fmt.Errorf("error reading file into memory: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	if CLI.CacheInMemory {
		// Wrap reader in own struct that implements io.Reader
		noSeek := &noSeeker{
			r: inf,
		}
		reader = noSeek
	}

	// capture start time
	start := time.Now()

	// extract file
	if err := extractionFunction(ctx, reader, tmpExtract); err != nil {
		return -1, fmt.Errorf("error performing extraction with %s: %w", libraryName, err)
	}
	duration := time.Since(start)

	// capture duration
	logger.Debug("extraction finished", "libraryName", libraryName, "filename", filename, "duration", fmt.Sprintf("%dms", duration.Milliseconds()))
	return duration.Milliseconds(), nil

}

// min returns the minimum value of the given slice
func min(slice []int64) int64 {
	min := int64(math.MaxInt64)
	for _, value := range slice {
		if value < min {
			min = value
		}
	}
	return min
}

// max returns the maximum value of the given slice
func max(slice []int64) int64 {
	max := int64(math.MinInt64)
	for _, value := range slice {
		if value > max {
			max = value
		}
	}
	return max
}

// avg returns the average of the given slice
func avg(slice []int64) int64 {
	var sum int64
	for _, value := range slice {
		sum += value
	}
	return sum / int64(len(slice))
}

// std returns the standard deviation of the given slice
func std(slice []int64) float64 {
	avg := avg(slice)
	var sum float64
	for _, value := range slice {
		sum += math.Pow(float64(value)-float64(avg), 2)
	}
	return math.Sqrt(sum / float64(len(slice)))
}

func unpackParallel(ctx context.Context, slugTarget string, body io.Reader) error {

	// reading from the TeeReader will write to the pipe
	pipeRead, pipeWrite := io.Pipe()
	tee := io.TeeReader(body, pipeWrite)

	eg := &errgroup.Group{}

	eg.Go(func() error {
		// ++++++++++++++++++++++++++++++++
		// old implementation of unpacking
		// ++++++++++++++++++++++++++++++++

		// read from the TeeReader which writes to the first pipe
		defer pipeWrite.Close()

		// Unpack the archive into the temporary directory
		err := slug.Unpack(tee, slugTarget)
		if err != nil {
			return err
		}

		return nil
	})

	goExtractTarget, _ := os.MkdirTemp("", "go-extract-*")
	defer os.RemoveAll(goExtractTarget)

	eg.Go(func() error {
		// ++++++++++++++++++++++++++++++++
		// new implementation of unpacking
		// ++++++++++++++++++++++++++++++++

		// read from the pipe as the other goroutine writes to it
		defer pipeRead.Close()

		// unpack archive into temporary telemetry dir
		return extract.Unpack(ctx, pipeRead, goExtractTarget, unpackConfigGoExtract)
	})

	if err := eg.Wait(); err != nil {
		slugTarget = ""
		return err
	}

	// compare the two directories
	return compareDirectories(slugTarget, goExtractTarget)
}

// compareFiles compares the two directories
func compareDirectories(slugDir string, goExtractTarget string) error {

	slugFiles := make(map[string]os.FileInfo)
	goExtractFiles := make(map[string]os.FileInfo)

	// get all files in the directory
	filepath.Walk(slugDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "Error walking directory")
		}
		slugFiles[strings.TrimPrefix(path, slugDir)] = info
		return nil
	})

	// get all files in the directory
	filepath.Walk(goExtractTarget, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "Error walking directory")
		}
		goExtractFiles[strings.TrimPrefix(path, goExtractTarget)] = info
		return nil
	})

	// compare the two directories
	for path, info := range slugFiles {
		if goExtractFiles[path] == nil {
			return fmt.Errorf("file %s not found in go-extract target", path)
		}
		if info.Size() != goExtractFiles[path].Size() {
			return fmt.Errorf("file %s has different size in go-extract target", path)
		}
	}

	return nil
}
