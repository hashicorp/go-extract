package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/go-extract"
	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/metrics"
)

// CLI are the cli parameters for go-extract binary
type CLI struct {
	Archive                    string           `arg:"" name:"archive" help:"Path to archive. (\"-\" for STDIN)" type:"existing file"`
	ContinueOnError            bool             `short:"C" help:"Continue extraction on error."`
	ContinueOnUnsupportedFiles bool             `short:"S" help:"Skip extraction of unsupported files."`
	CreateDestination          bool             `short:"c" help:"Create destination directory if it does not exist."`
	DenySymlinks               bool             `short:"D" help:"Deny symlink extraction."`
	Destination                string           `arg:"" name:"destination" default:"." help:"Output directory/file."`
	FollowSymlinks             bool             `short:"F" help:"[Dangerous!] Follow symlinks to directories during extraction."`
	MaxFiles                   int64            `optional:"" default:"1000" help:"Maximum files that are extracted before stop. (disable check: -1)"`
	MaxExtractionSize          int64            `optional:"" default:"1073741824" help:"Maximum extraction size that allowed is (in bytes). (disable check: -1)"`
	MaxExtractionTime          int64            `optional:"" default:"60" help:"Maximum time that an extraction should take (in seconds). (disable check: -1)"`
	MaxInputSize               int64            `optional:"" default:"1073741824" help:"Maximum input size that allowed is (in bytes). (disable check: -1)"`
	Metrics                    bool             `short:"M" optional:"" default:"false" help:"Print metrics to log after extraction."`
	NoUntarAfterDecompression  bool             `short:"N" optional:"" default:"false" help:"Disable combined extraction of tar.gz."`
	Overwrite                  bool             `short:"O" help:"Overwrite if exist."`
	Pattern                    []string         `short:"P" optional:"" name:"pattern" help:"Extracted objects need to match shell file name pattern."`
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
			"version": fmt.Sprintf("%s (%s), commit %s, built at %s", filepath.Base(os.Args[0]), version, commit, date),
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

	// setup metrics hook
	metricsToLog := func(ctx context.Context, metrics *metrics.Metrics) {
		if cli.Metrics {
			logger.Info("extraction finished", "metrics", metrics)
		}
	}

	// process cli params
	config := config.NewConfig(
		config.WithContinueOnError(cli.ContinueOnError),
		config.WithContinueOnUnsupportedFiles(cli.ContinueOnUnsupportedFiles),
		config.WithCreateDestination(cli.CreateDestination),
		config.WithDenySymlinkExtraction(cli.DenySymlinks),
		config.WithFollowSymlinks(cli.FollowSymlinks),
		config.WithLogger(logger),
		config.WithMaxExtractionSize(cli.MaxExtractionSize),
		config.WithMaxFiles(cli.MaxFiles),
		config.WithMaxInputSize(cli.MaxInputSize),
		config.WithMetricsHook(metricsToLog),
		config.WithNoUntarAfterDecompression(cli.NoUntarAfterDecompression),
		config.WithOverwrite(cli.Overwrite),
		config.WithPatterns(cli.Pattern...),
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
		log.Println(fmt.Errorf("error during extraction: %s", err))
		os.Exit(-1)
	}
}
