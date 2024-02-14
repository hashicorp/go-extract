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
