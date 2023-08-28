package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/go-extract"
	"github.com/hashicorp/go-extract/config"
)

type CLI struct {
	Archive           string `arg:"" name:"input" help:"Path to csv or Epic issue id." type:"file"`
	Force             bool   `short:"F" help:"Force extraction and overwrite if exist"`
	MaxFiles          int64  `optional:"" default:"1000" help:"Maximum files that are extracted before stop"`
	MaxFileSize       int64  `optional:"" default:"1073741824" help:"Maximum file size that allowed is (in bytes)"`
	MaxExtractionTime int64  `optional:"" default:"60" help:"Maximum time that an extraction should take (in seconds)"`
	Output            string `short:"o" optional:"" default:"." help:"Output directory"`
	Verbose           bool   `short:"v" optional:"" help:"Verbose logging."`
	Version           bool   `short:"V" optional:"" help:"Print release version information."`
}

func Run(version, commit, date string) {
	ctx := context.Background()
	var cli CLI
	kong.Parse(&cli)

	// Check for verbose output
	if cli.Verbose {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(io.Discard)
	}

	// check if version information is requested
	if cli.Version {
		fmt.Printf("sat (%s), commit %s, built at %s\n", version, commit, date)
		return
	}

	// process cli params
	config := config.NewConfig(
		config.WithMaxExtractionTime(cli.MaxExtractionTime),
		config.WithMaxFiles(cli.MaxFiles),
		config.WithMaxFileSize(cli.MaxFileSize),
		config.WithOverwrite(cli.Force),
	)
	extractOptions := []extract.ExtractorOption{
		extract.WithConfig(config),
	}

	// open archive
	archive, err := os.Open(cli.Archive)
	if err != nil {
		panic(err)
	}

	// extract archive
	if err := extract.Unpack(ctx, archive, cli.Output, extractOptions...); err != nil {
		log.Println(fmt.Errorf("error during extraction: %w", err))
		os.Exit(-1)
	}
}
