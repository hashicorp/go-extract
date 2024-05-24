package extractor

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// createFile is a wrapper around the target.CreateFile function
func createFile(t target.Target, dstDir string, name string, reader io.Reader, mode fs.FileMode, config *config.Config) error {
	return t.CreateSafeFile(dstDir, name, reader, mode, config)
}

// createDir is a wrapper around the target.CreateDir function
func createDir(t target.Target, dstDir string, dirName string, mode fs.FileMode, config *config.Config) error {
	return t.CreateSafeDir(dstDir, dirName, mode, config)
}

// createSymlink is a wrapper around the target.CreateSymlink function
func createSymlink(t target.Target, dstDir string, name string, linkTarget string, config *config.Config) error {
	return t.CreateSafeSymlink(dstDir, name, linkTarget, config)
}
