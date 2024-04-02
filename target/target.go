package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {

	// CreateSafeFile is used to create a file in directory dstDir, with name and reader as content
	CreateSafeFile(config *config.Config, dstDir string, name string, reader io.Reader, perm fs.FileMode) error

	// CreateSafeDir creates dirName in dstDir
	CreateSafeDir(config *config.Config, dstDir string, dirName string, perm fs.FileMode) error

	// CreateSafeSymlink creates symlink name with destination linkTarget in dstDir
	CreateSafeSymlink(config *config.Config, dstDir string, name string, linkTarget string) error
}
