package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/go-extract/pkg/extract"
)

type CLI struct {
	Archive string `arg:"" name:"input" help:"Path to csv or Epic issue id." type:"file"`
	Force   bool   `short:"F" help:"Force extraction and overwrite if exist"`
	Output  string `short:"o" optional:"" default:"." help:"Output directory"`
	Verbose bool   `short:"v" optional:"" help:"Verbose logging."`
	Version bool   `short:"V" optional:"" help:"Print release version information."`
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

	// extract archive
	if err := extract.Extract(ctx, cli.Archive, cli.Output); err != nil {
		log.Printf("error during extraction: %w", err)
		os.Exit(-1)
	}
}
