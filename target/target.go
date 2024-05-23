package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {

	// CreateSafeFile creates a file with a specified name and content within a given directory.
	// The function takes a configuration object, a destination directory, a name for the file,
	// a reader for the file content, and a file mode.
	//
	// The dstDir parameter specifies the directory where the file should be created. If dstDir is empty,
	// the file is created in the current working directory. If dstDir does not exist and config.CreateDestination()
	// returns true, the function may create the directory with permissions specified by config.CustomCreateDirMode().
	//
	// The name parameter specifies the name of the file. If the name contains path traversal (e.g., "../"),
	// the function returns an error to prevent the creation of files outside of the intended directory.
	//
	// The reader parameter provides the content for the file.
	//
	// The mode parameter specifies the file mode that should be set on the file.
	//
	// If the path to the file (excluding dstDir) does not exist, the function may create the necessary directories
	// with permissions specified by config.CustomCreateDirMode().
	//
	// The function returns an error if there's a problem creating the file or the necessary directories.
	// If the function completes successfully, it returns nil.
	CreateSafeFile(config *config.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode) error

	// CreateSafeDir creates a directory with a specified name within a given directory.
	// The function takes a configuration object, a destination directory, a name for the directory, and a file mode.
	//
	// The dstDir parameter specifies the directory where the new directory should be created. If dstDir is empty,
	// the directory is created in the current working directory. If dstDir does not exist and config.CreateDestination()
	// returns true, the function may create the directory with permissions specified by config.CustomCreateDirMode().
	//
	// The dirName parameter specifies the name of the new directory. If the dirName contains path traversal (e.g., "../"),
	// the function returns an error to prevent the creation of directories outside of the intended directory.
	//
	// The mode parameter specifies the file mode that should be set on the new directory.
	//
	// If the path to the new directory (excluding dstDir) does not exist, the function may create the necessary directories
	// with permissions specified by config.CustomCreateDirMode().
	//
	// The function returns an error if there's a problem creating the directory or the necessary directories.
	// If the function completes successfully, it returns nil.
	CreateSafeDir(config *config.Config, dstDir string, dirName string, mode fs.FileMode) error

	// CreateSafeSymlink creates a symbolic link with a specified name and target within a given directory.
	// The function takes a configuration object, a destination directory, a name for the symlink, and a target for the symlink.
	//
	// The dstDir parameter specifies the directory where the symlink should be created. If dstDir is empty,
	// the symlink is created in the current working directory. If dstDir does not exist and config.CreateDestination()
	// returns true, the function may create the directory with permissions specified by config.CustomCreateDirMode().
	//
	// The name parameter specifies the name of the symlink. If the name contains path traversal (e.g., "../"),
	// the function returns an error to prevent the creation of symlinks outside of the intended directory.
	//
	// The linkTarget parameter specifies the target of the symlink. This is the file or directory that the symlink will point to.
	//
	// If the path to the symlink (excluding dstDir) does not exist, the function may create the necessary directories
	// with permissions specified by config.CustomCreateDirMode().
	//
	// The function returns an error if there's a problem creating the symlink or the necessary directories.
	// If the function completes successfully, it returns nil.
	CreateSafeSymlink(config *config.Config, dstDir string, name string, linkTarget string) error
}
