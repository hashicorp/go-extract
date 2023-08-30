package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {

	// CreateSafeFile is used to create a file in directory dstDir, with name and reader as content
	CreateSafeFile(dstDir string, name string, reader io.Reader, mode fs.FileMode) error

	// CreateSafeDir creates dirName in dstDir
	CreateSafeDir(dstDir string, dirName string) error

	// CreateSafeSymlink creates symlink name with destination linkTarget in dstDir
	CreateSafeSymlink(dstDir string, name string, linkTarget string) error

	// SetConfig sets the config for a target
	SetConfig(config *config.Config)
}
