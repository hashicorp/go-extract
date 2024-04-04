package target

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
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

// CreateSafeDir creates newDir in dstBase and checks for path traversal in directory name
func (o *OS) CreateDir(name string, perm fs.FileMode) error {

	if err := os.MkdirAll(name, perm); err != nil {
		return fmt.Errorf("failed to create directory (%s)", err)
	}

	return nil
}

// CreateSafeFile creates newFileName in dstBase with content from reader and file
// headers as provided in mode
func (o *OS) CreateFile(name string, reader io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {

	// Check for file existence//overwrite
	if _, err := os.Lstat(name); !os.IsNotExist(err) {
		if !overwrite {
			return 0, fmt.Errorf("file already exists")
		}
	}
	// TODO: check behaviour for symlinks

	// create dst file
	dstFile, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		dstFile.Close()
	}()

	// encapsulate reader with limit reader
	if maxSize >= 0 {
		limitedWriter := NewLimitErrorWriter(dstFile, maxSize)
		return io.Copy(limitedWriter, reader)
	}

	// write data straight to file
	return io.Copy(dstFile, reader)
}

// CreateSymlink creates in dstBase a symlink newLinkName with destination linkTarget
func (o *OS) CreateSymlink(name string, target string, overwrite bool) error {

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(name); !os.IsNotExist(err) {
		if !overwrite {
			return fmt.Errorf("file already exist")
		}

		if err := os.Remove(name); err != nil {
			return fmt.Errorf("failed to overwrite: %w", err)
		}
	}

	// create link
	if err := os.Symlink(target, name); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// Lstat returns the FileInfo of the file at path
func (o *OS) Lstat(path string) (fs.FileInfo, error) {
	return os.Lstat(path)
}

func GetStartOfAbsolutePath(path string) string {

	// check absolute path for link target on unix
	if strings.HasPrefix(path, "/") {
		return fmt.Sprintf("%s%s", "/", GetStartOfAbsolutePath(path[1:]))
	}

	// check absolute path for link target on unix
	if strings.HasPrefix(path, `\`) {
		return fmt.Sprintf("%s%s", `\`, GetStartOfAbsolutePath(path[1:]))
	}

	// check absolute path for link target on windows
	if p := []rune(path); len(p) > 2 && p[1] == rune(':') {
		return fmt.Sprintf("%s%s", path[0:3], GetStartOfAbsolutePath(path[3:]))
	}

	return ""
}
