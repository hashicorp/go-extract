package extractor

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// createFile is a wrapper around the target.CreateFile function
func createFile(t target.Target, dst string, name string, src io.Reader, mode fs.FileMode, cfg *config.Config) error {
	return t.CreateSafeFile(dst, name, src, mode, cfg)
}

// createDir is a wrapper around the target.CreateDir function
func createDir(t target.Target, dst string, name string, mode fs.FileMode, cfg *config.Config) error {
	return t.CreateSafeDir(dst, name, mode, cfg)
}

// createSymlink is a wrapper around the target.CreateSymlink function
func createSymlink(t target.Target, dst string, name string, linkTarget string, cfg *config.Config) error {
	return t.CreateSafeSymlink(dst, name, linkTarget, cfg)
}
