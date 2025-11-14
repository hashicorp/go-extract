// Copyright IBM Corp. 2023, 2025

package main

import "github.com/hashicorp/go-extract/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// main start go-extract cli `extract`
func main() {
	cmd.Run(version, commit, date)
}
