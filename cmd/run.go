// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	extract "github.com/hashicorp/go-extract"
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
	InsecureTraverseSymlinks   bool             `help:"Traverse symlinks to directories during extraction."`
	MaxFiles                   int64            `optional:"" default:"${default_max_files}" help:"Maximum files (including folder and symlinks) that are extracted before stop. (disable check: -1)"`
	MaxExtractionSize          int64            `optional:"" default:"${default_max_extraction_size}" help:"Maximum extraction size that allowed is (in bytes). (disable check: -1)"`
	MaxExtractionTime          int64            `optional:"" default:"${default_max_extraction_time}" help:"Maximum time that an extraction should take (in seconds). (disable check: -1)"`
	MaxInputSize               int64            `optional:"" default:"${default_max_input_size}" help:"Maximum input size that allowed is (in bytes). (disable check: -1)"`
	NoUntarAfterDecompression  bool             `short:"N" optional:"" default:"false" help:"Disable combined extraction of tar.gz."`
	Overwrite                  bool             `short:"O" help:"Overwrite if exist."`
	Pattern                    []string         `short:"P" optional:"" name:"pattern" help:"Extracted objects need to match shell file name pattern."`
	PreserveFileAttributes     bool             `short:"p" help:"Preserve file attributes from archive (access and modification time & file permissions)."`
	PreserveOwner              bool             `short:"o" help:"Preserve owner and group of files from archive (only root/uid:0 on unix systems for tar files)."`
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
			"version":                     fmt.Sprintf("%s (%s), commit %s, built at %s", filepath.Base(os.Args[0]), version, commit, date),
			"valid_types":                 "7z, br, bz2, gz, lz4, rar, sz, tar, tgz, xz, zip, zst, zz",
			"default_type":                "",                          // default is empty, but needs to be set to avoid kong error
			"default_max_extraction_size": strconv.Itoa(1 << (10 * 3)), // 1GB
			"default_max_files":           strconv.Itoa(100000),        // 100k files
			"default_max_input_size":      strconv.Itoa(1 << (10 * 3)), // 1GB
			"default_max_extraction_time": strconv.Itoa(60),            // 60 seconds
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
	telemetryDataToLog := func(ctx context.Context, td *extract.TelemetryData) {
		if cli.Telemetry {
			logger.Info("extraction finished", "telemetryData", td)
		}
	}

	// process cli params
	config := extract.NewConfig(
		extract.WithContinueOnError(cli.ContinueOnError),
		extract.WithContinueOnUnsupportedFiles(cli.ContinueOnUnsupportedFiles),
		extract.WithCreateDestination(cli.CreateDestination),
		extract.WithCustomCreateDirMode(toFileMode(cli.CustomCreateDirMode)),
		extract.WithCustomDecompressFileMode(toFileMode(cli.CustomDecompressFileMode)),
		extract.WithDenySymlinkExtraction(cli.DenySymlinks),
		extract.WithExtractType(cli.Type),
		extract.WithInsecureTraverseSymlinks(cli.InsecureTraverseSymlinks),
		extract.WithLogger(logger),
		extract.WithMaxExtractionSize(cli.MaxExtractionSize),
		extract.WithMaxFiles(cli.MaxFiles),
		extract.WithMaxInputSize(cli.MaxInputSize),
		extract.WithNoUntarAfterDecompression(cli.NoUntarAfterDecompression),
		extract.WithOverwrite(cli.Overwrite),
		extract.WithPatterns(cli.Pattern...),
		extract.WithPreserveFileAttributes(cli.PreserveFileAttributes),
		extract.WithPreserveOwner(cli.PreserveOwner),
		extract.WithTelemetryHook(telemetryDataToLog),
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
	if err := extract.Unpack(ctx, cli.Destination, archive, config); err != nil {
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
