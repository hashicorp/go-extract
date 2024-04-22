package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {

	// CreateSafeFile is used to create a file in directory dstDir, with name and reader as content.
	// dstDir is the directory where the file should be created. If dstDir is empty, the file is meant to
	// be created in the current working directory. If dstDir does not exist and config.CreateDestination() returns
	// true, it might be created with the config.CustomCreateDirMode() by the implementation. The mode parameter is the
	// file mode that should be set on the file.
	// If the path of the file contains path traversal, an error should be returned. If the path *to the file* (not dstDir) does not
	// exist, the directories should be created with the config.CustomCreateDirMode() by the implementation.
	CreateSafeFile(config *config.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode) error

	// CreateSafeDir creates dirName in dstDir. If dstDir is empty, the directory is meant to be created in the current working directory.
	// If dstDir does not exist and config.CreateDestination() returns true, it might be created with the config.CustomCreateDirMode() by the
	// implementation. The mode parameter is the file mode that should be set on the directory.
	CreateSafeDir(config *config.Config, dstDir string, dirName string, mode fs.FileMode) error

	// CreateSafeSymlink creates symlink name with destination linkTarget in dstDir.
	// dstDir is the directory where the link should be created. If dstDir is empty, the link is meant to
	// be created in the current working directory. If dstDir does not exist and config.CreateDestination() returns
	// true, it might be created with the config.CustomCreateDirMode() by the implementation.
	// If the path the link name contains path traversal, an error should be returned. If the path *to the link* (not dstDir) does not
	// exist, the directories should be created with the config.CustomCreateDirMode() by the implementation.
	CreateSafeSymlink(config *config.Config, dstDir string, name string, linkTarget string) error
}
