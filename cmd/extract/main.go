package main

import "github.com/hashicorp/go-extract/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// main start the security assessment tool
func main() {
	cmd.Run(version, commit, date)
}
