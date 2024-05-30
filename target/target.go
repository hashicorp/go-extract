package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {

	// CreateSafeFile creates a file with a specified name and content within a given dst directory.
	// The function takes a configuration object cfg, a destination directory dst, a name for the file,
	// a reader src for the file content, and a file mode.
	//
	// The dst parameter specifies the directory where the file should be created. If dst is empty,
	// the file is created in the current working directory. If dst does not exist and cfg.CreateDestination()
	// returns true, the function may create the directory with permissions specified by cfg.CustomCreateDirMode().
	//
	// The name parameter specifies the name of the file. If the name contains path traversal (e.g., "../"),
	// the function returns an error to prevent the creation of files outside of the intended dst directory.
	//
	// The reader parameter provides the content for the file.
	//
	// The mode parameter specifies the file mode that should be set on the file.
	//
	// If the path to the file (excluding dst) does not exist, the function may create the necessary directories
	// with permissions specified by cfg.CustomCreateDirMode().
	//
	// The function returns an error if there's a problem creating the file or the necessary directories.
	// If the function completes successfully, it returns nil.
	CreateSafeFile(dst string, name string, src io.Reader, mode fs.FileMode, cfg *config.Config) error

	// CreateSafeDir creates a directory with a specified name within a given directory.
	// The function takes a configuration object cfg, a destination directory dst, a name for the directory, and a file mode.
	//
	// The dst parameter specifies the directory where the new directory should be created. If dst is empty,
	// the directory is created in the current working directory. If dst does not exist and cfg.CreateDestination()
	// returns true, the function may create the directory with permissions specified by cfg.CustomCreateDirMode().
	//
	// The name parameter specifies the name of the new directory. If the name contains path traversal (e.g., "../"),
	// the function returns an error to prevent the creation of directories outside of the intended directory.
	//
	// The mode parameter specifies the file mode that should be set on the new directory.
	//
	// If the path to the new directory (excluding dst) does not exist, the function may create the necessary directories
	// with permissions specified by cfg.CustomCreateDirMode().
	//
	// The function returns an error if there's a problem creating the directory or the necessary directories.
	// If the function completes successfully, it returns nil.
	CreateSafeDir(dst string, name string, mode fs.FileMode, cfg *config.Config) error

	// CreateSafeSymlink creates a symbolic link with a specified name and target within a given directory.
	// The function takes a configuration object, a destination directory, a name for the symlink, and a target for the symlink.
	//
	// The dst parameter specifies the directory where the symlink should be created. If dst is empty,
	// the symlink is created in the current working directory. If dst does not exist and cfg.CreateDestination()
	// returns true, the function may create the directory with permissions specified by cfg.CustomCreateDirMode().
	//
	// The name parameter specifies the name of the symlink. If the name contains path traversal (e.g., "../"),
	// the function returns an error to prevent the creation of symlinks outside of the intended dst directory.
	//
	// The target parameter specifies the target of the symlink. This is the file or directory that the symlink will point to.
	//
	// If the path to the symlink (excluding dst) does not exist, the function may create the necessary directories
	// with permissions specified by cfg.CustomCreateDirMode().
	//
	// The function returns an error if there's a problem creating the symlink or the necessary directories.
	// If the function completes successfully, it returns nil.
	CreateSafeSymlink(dst string, name string, target string, cfg *config.Config) error
}
