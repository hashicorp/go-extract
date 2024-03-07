package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// now is a function point that returns time.Now to the caller.
var now = time.Now

// unpackTarget is the target that is used for extraction
var unpackTarget target.Target

// SeekerReaderAt is a struct that combines the io.ReaderAt and io.Seeker interfaces
type SeekerReaderAt interface {
	io.ReaderAt
	io.Seeker
}

// determineOutputName determines the output name and directory for the extracted content
func determineOutputName(dst string, src io.Reader) (string, string) {

	// check if dst is specified and not a directory
	if dst != "." && dst != "" {
		if stat, err := os.Stat(dst); os.IsNotExist(err) || stat.Mode()&fs.ModeDir == 0 {
			return filepath.Dir(dst), filepath.Base(dst)
		}
	}

	// check if src is a file and the filename is ending with the suffix
	// remove the suffix from the filename and use it as output name
	if f, ok := src.(*os.File); ok {

		name := filepath.Base(f.Name())
		newName := strings.TrimSuffix(name, filepath.Ext(name))
		if name != newName && newName != "" {
			return dst, newName
		}

		// if the filename is not ending with the suffix, use the suffix as output name
		return dst, fmt.Sprintf("%s.decompressed", newName)
	}
	return dst, "goextract-decompressed-content"
}

// limitReader ensures that the input size is limited and the input size is captured
// remark: this preparation is located in the extractor package so that the
// different extractor engines can be used independently and keep their
// functionality.
func limitReader(src io.Reader, c *config.Config) io.Reader {
	ler := config.NewLimitErrorReader(src, c.MaxInputSize())
	c.AddMetricsProcessor(func(ctx context.Context, m *config.Metrics) {
		m.InputSize = int64(ler.ReadBytes())
	})
	return ler
}

// checkPatterns checks if the given path matches any of the given patterns.
// If no patterns are given, the function returns true.
func checkPatterns(patterns []string, path string) (bool, error) {

	// no patterns given
	if len(patterns) == 0 {
		return true, nil
	}

	// check if path matches any pattern
	for _, pattern := range patterns {
		if match, err := filepath.Match(pattern, path); err != nil {
			return false, fmt.Errorf("failed to match pattern: %s", err)
		} else if match {
			return true, nil
		}
	}
	return false, nil
}

// captureExtractionDuration ensures that the extraction duration is captured
func captureExtractionDuration(c *config.Config) {
	start := now()
	c.AddMetricsProcessor(func(ctx context.Context, m *config.Metrics) {
		m.ExtractionDuration = time.Since(start) // capture execution time
	})
}

// UnpackFunc is a function that extracts the contents from src and extracts them to dst.
type UnpackFunc func(context.Context, io.Reader, string, *config.Config) error

// HeaderCheck is a function that checks if the given header matches the expected magic bytes.
type HeaderCheck func([]byte) bool

// AvailableExtractors is collection of new extractor functions with
// the required magic bytes and potential offset
var AvailableExtractors = []struct {
	Unpacker      UnpackFunc
	HeaderCheck   HeaderCheck
	MagicBytes    [][]byte
	Offset        int
	FileExtension string
}{
	{
		Unpacker:      UnpackTar,
		HeaderCheck:   IsTar,
		MagicBytes:    magicBytesTar,
		Offset:        offsetTar,
		FileExtension: fileExtensionTar,
	},
	{
		Unpacker:      UnpackZip,
		HeaderCheck:   IsZip,
		MagicBytes:    magicBytesZIP,
		FileExtension: fileExtensionZIP,
	},
	{
		Unpacker:      UnpackGZip,
		HeaderCheck:   IsGZip,
		MagicBytes:    magicBytesGZip,
		FileExtension: fileExtensionGZip,
	},
	{
		Unpacker:      unpackBrotli,
		HeaderCheck:   IsBrotli,
		MagicBytes:    magicBytesBrotli,
		FileExtension: fileExtensionBrotli,
	},
	{
		Unpacker:      UnpackBzip2,
		HeaderCheck:   IsBzip2,
		MagicBytes:    magicBytesBzip2,
		FileExtension: fileExtensionBzip2,
	},
}

var MaxHeaderLength int

// init calculates the maximum header length
func init() {
	for _, ex := range AvailableExtractors {
		needs := ex.Offset
		for _, mb := range ex.MagicBytes {
			if len(mb)+ex.Offset > needs {
				needs = len(mb) + ex.Offset
			}
		}
		if needs > MaxHeaderLength {
			MaxHeaderLength = needs
		}
	}

	// set default target
	unpackTarget = target.NewOS()
}

func matchesMagicBytes(data []byte, offset int, magicBytes [][]byte) bool {
	// check all possible magic bytes until match is found
	for _, mb := range magicBytes {
		// check if header is long enough
		if offset+len(mb) > len(data) {
			continue
		}

		// check for byte match
		if bytes.Equal(mb, data[offset:offset+len(mb)]) {
			return true
		}
	}

	// no match found
	return false
}

// handleError increases the error counter, sets the latest error and
// decides if extraction should continue.
func handleError(c *config.Config, metrics *config.Metrics, msg string, err error) error {

	// increase error counter and set error
	metrics.ExtractionErrors++
	metrics.LastExtractionError = fmt.Errorf("%s: %s", msg, err)

	// do not end on error
	if c.ContinueOnError() {
		c.Logger().Error(msg, "error", err)
		return nil
	}

	// end extraction on error
	return metrics.LastExtractionError
}
