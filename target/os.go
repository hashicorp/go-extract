package target

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// OS is the struct type that holds all information for interacting with the filesystem
type OS struct {
}

// NewOS creates a new Os and applies provided options from opts
func NewOS() *OS {
	// create object
	os := &OS{}
	return os
}

// CreateDir creates newDir in dstBase and checks for path traversal in directory name.
// If dstBase is empty, the directory will be created in the current working directory. If dstBase
// does not exist and config.CreateDestination() returns true, it will be created with the
// config.CustomCreateDirMode(). The mode parameter is the file mode that should be set on the directory.
// If the directory already exists, the mode will be set on the directory.
func (o *OS) CreateDir(dstBase string, newDir string, mode fs.FileMode) error {

	// create dirs
	fullPath := filepath.Join(dstBase, newDir)
	if err := os.MkdirAll(fullPath, mode.Perm()); err != nil {
		return fmt.Errorf("failed to create directory (%s)", err)
	}

	return nil
}

// CreateFile creates newFileName in dstBase with content from reader and file
// headers as provided in mode. If dstBase is empty, the file will be created in the current
// working directory. If dstBase does not exist and config.CreateDestination() returns true, it will be created with the
// config.CustomCreateDirMode(). The mode parameter is the file mode that is set on the file.
// If the path of the file contains path traversal, an error should be returned. If the path *to the file* (not dstBase) does not
// exist, the directories is created with the config.CustomCreateDirMode() by the implementation.
func (o *OS) CreateFile(dstBase string, newFileName string, reader io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {

	// Check for path validity and if file existence+overwrite
	targetFile := filepath.Join(dstBase, newFileName)
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {

		// something wrong with path
		if err != nil {
			return 0, fmt.Errorf("invalid path (%s): %s", newFileName, err)
		}

		// check for overwrite
		if !overwrite {
			return 0, fmt.Errorf("file already exists (%s)", newFileName)
		}
	}

	// create dst file
	dstFile, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return 0, fmt.Errorf("failed to create file (%s)", err)
	}
	defer func() {
		dstFile.Close()
	}()

	// write data to file
	writer := limitWriter(dstFile, maxSize)
	n, err := io.Copy(writer, reader)
	if err != nil {
		return n, fmt.Errorf("failed to write file (%s)", err)
	}

	return n, err
}

// CreateSymlink creates in dstBase a symlink newLinkName with destination linkTarget.
// If dstBase is empty, the symlink is created in the current working directory. If dstBase
// does not exist and config.CreateDestination() returns true, it will be created with the
// config.CustomCreateDirMode(). If the path of the symlink contains path traversal, an error
// is returned. If the path *to the symlink* (not dstBase) does not exist, the directories
// is created with the config.CustomCreateDirMode(). If the symlink already exists and
// config.Overwrite() returns false, an error is returned.
func (o *OS) CreateSymlink(dstBase string, name string, target string, overwrite bool) error {

	// new file
	targetFile := filepath.Join(dstBase, name)

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		if !overwrite {
			return fmt.Errorf("symlink already exist (%s)", name)
		}

		// delete existing link
		if err := os.Remove(targetFile); err != nil {
			return fmt.Errorf("failed to remove existing symlink (%s)", err)
		}
	}

	// create link
	if err := os.Symlink(target, targetFile); err != nil {
		return fmt.Errorf("failed to create symlink (%s)", err)
	}

	return nil
}

// Lstat returns file information for the specified file or directory.
func (o *OS) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(name)
}

// Readlink returns the target of a symbolic link.
func (o *OS) Readlink(name string) (string, error) {
	return os.Readlink(name)
}
