package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/go-extract"
	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

// CLI are the cli parameters for go-extract binary
type CLI struct {
	Archive                    string           `arg:"" name:"archive" help:"Path to archive. (\"-\" for STDIN)" type:"existing file"`
	ContinueOnError            bool             `short:"C" help:"Continue extraction on error."`
	ContinueOnUnsupportedFiles bool             `short:"S" help:"Skip extraction of unsupported files."`
	CreateDestination          bool             `short:"c" help:"Create destination directory if it does not exist."`
	CustomCreateDirMode        int              `optional:"" default:"750" help:"File mode for created directories, which are not listed in the archive. (respecting umask)"`
	CustomDecompressFileMode   int              `optional:"" default:"640" help:"File mode for decompressed files. (respecting umask)"`
	DenySymlinks               bool             `short:"D" help:"Deny symlink extraction."`
	Destination                string           `arg:"" name:"destination" default:"." help:"Output directory/file."`
	FollowSymlinks             bool             `short:"F" help:"[Dangerous!] Follow symlinks to directories during extraction."`
	MaxFiles                   int64            `optional:"" default:"1000" help:"Maximum files (including folder and symlinks) that are extracted before stop. (disable check: -1)"`
	MaxExtractionSize          int64            `optional:"" default:"1073741824" help:"Maximum extraction size that allowed is (in bytes). (disable check: -1)"`
	MaxExtractionTime          int64            `optional:"" default:"60" help:"Maximum time that an extraction should take (in seconds). (disable check: -1)"`
	MaxInputSize               int64            `optional:"" default:"1073741824" help:"Maximum input size that allowed is (in bytes). (disable check: -1)"`
	NoUntarAfterDecompression  bool             `short:"N" optional:"" default:"false" help:"Disable combined extraction of tar.gz."`
	Overwrite                  bool             `short:"O" help:"Overwrite if exist."`
	Pattern                    []string         `short:"P" optional:"" name:"pattern" help:"Extracted objects need to match shell file name pattern."`
	Telemetry                  bool             `short:"T" optional:"" default:"false" help:"Print telemetry data to log after extraction."`
	Type                       string           `short:"t" optional:"" default:"${default_type}" name:"type" help:"Type of archive. (${valid_types})"`
	Verbose                    bool             `short:"v" optional:"" help:"Verbose logging."`
	Version                    kong.VersionFlag `short:"V" optional:"" help:"Print release version information."`
}

// Run the entrypoint into go-extract as a cli tool
func Run(version, commit, date string) {
	ctx := context.Background()
	var cli CLI
	kong.Parse(&cli,
		kong.Description("A secure extraction utility"),
		kong.UsageOnError(),
		kong.Vars{
			"version":      fmt.Sprintf("%s (%s), commit %s, built at %s", filepath.Base(os.Args[0]), version, commit, date),
			"valid_types":  extract.ValidTypes(),
			"default_type": "", // default is empty, but needs to be set to avoid kong error
		},
	)

	// Check for verbose output
	logLevel := slog.LevelError
	if cli.Verbose {
		logLevel = slog.LevelDebug
	}

	// setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// setup telemetry hook
	telemetryDataToLog := func(ctx context.Context, td *telemetry.Data) {
		if cli.Telemetry {
			logger.Info("extraction finished", "telemetryData", td)
		}
	}

	// process cli params
	config := config.NewConfig(
		config.WithContinueOnError(cli.ContinueOnError),
		config.WithContinueOnUnsupportedFiles(cli.ContinueOnUnsupportedFiles),
		config.WithCreateDestination(cli.CreateDestination),
		config.WithCustomCreateDirMode(toFileMode(cli.CustomCreateDirMode)),
		config.WithCustomDecompressFileMode(toFileMode(cli.CustomDecompressFileMode)),
		config.WithDenySymlinkExtraction(cli.DenySymlinks),
		config.WithExtractType(cli.Type),
		config.WithFollowSymlinks(cli.FollowSymlinks),
		config.WithLogger(logger),
		config.WithMaxExtractionSize(cli.MaxExtractionSize),
		config.WithMaxFiles(cli.MaxFiles),
		config.WithMaxInputSize(cli.MaxInputSize),
		config.WithNoUntarAfterDecompression(cli.NoUntarAfterDecompression),
		config.WithOverwrite(cli.Overwrite),
		config.WithPatterns(cli.Pattern...),
		config.WithTelemetryHook(telemetryDataToLog),
	)

	// open archive
	var archive io.Reader
	if cli.Archive == "-" {
		archive = bufio.NewReader(os.Stdin)
	} else {
		var err error
		if archive, err = os.Open(cli.Archive); err != nil {
			logger.Error("opening archive failed", "err", err)
			os.Exit(-1)
		} else {
			defer archive.(*os.File).Close()
		}
	}

	if cli.MaxExtractionTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), (time.Second * time.Duration(cli.MaxExtractionTime)))
		defer cancel()
	}

	// extract archive
	if err := extract.Unpack(ctx, archive, cli.Destination, config); err != nil {
		log.Println(fmt.Errorf("error during extraction: %w", err))
		os.Exit(-1)
	}
}

// asFileMode interprets the given decimal value as fs.FileMode
func toFileMode(v int) fs.FileMode {

	// convert to octal
	oct, _ := strconv.ParseInt(fmt.Sprintf("0%d", v), 8, 32)

	// return as fs.FileMode
	return fs.FileMode(oct)
}
