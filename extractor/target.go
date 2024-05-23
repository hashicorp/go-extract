package extractor

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// unpackTarget is the target that is used for extraction
var unpackTarget target.Target

// init initializes the unpackTarget
func init() {
	unpackTarget = target.NewOS()
}

// SetTarget sets the target for extraction
func SetTarget(t target.Target) {
	unpackTarget = t
}

// createFile is a wrapper around the target.CreateFile function
func createFile(config *config.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode) error {
	return unpackTarget.CreateSafeFile(config, dstDir, name, reader, mode)
}

// createDir is a wrapper around the target.CreateDir function
func createDir(config *config.Config, dstDir string, dirName string, mode fs.FileMode) error {
	return unpackTarget.CreateSafeDir(config, dstDir, dirName, mode)
}

// createSymlink is a wrapper around the target.CreateSymlink function
func createSymlink(config *config.Config, dstDir string, name string, target string) error {
	return unpackTarget.CreateSafeSymlink(config, dstDir, name, target)
}
