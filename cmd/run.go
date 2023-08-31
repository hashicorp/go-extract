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

// CLI are the cli parameters for go-extract binary
type CLI struct {
	Archive           string `arg:"" name:"archive" help:"Path to archive." type:"file"`
	Force             bool   `short:"f" help:"Force extraction and overwrite if exist"`
	MaxFiles          int64  `optional:"" default:"1000" help:"Maximum files that are extracted before stop"`
	MaxExtractionSize int64  `optional:"" default:"1073741824" help:"Maximum extraction size that allowed is (in bytes)"`
	MaxExtractionTime int64  `optional:"" default:"60" help:"Maximum time that an extraction should take (in seconds)"`
	Destination       string `arg:"" name:"destination" default:"." help:"Output directory/file"`
	Verbose           bool   `short:"v" optional:"" help:"Verbose logging."`
	Version           bool   `short:"V" optional:"" help:"Print release version information."`
}

// Run the entrypoint into go-extract as a cli tool
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
		config.WithMaxExtractionSize(cli.MaxExtractionSize),
		config.WithMaxFiles(cli.MaxFiles),
		config.WithForce(cli.Force),
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
	if err := extract.Unpack(ctx, archive, cli.Destination, extractOptions...); err != nil {
		log.Println(fmt.Errorf("error during extraction: %w", err))
		os.Exit(-1)
	}
}
