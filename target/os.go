package target

import (
	"fmt"
	"io"
	"io/fs"
	"os"
)

// OS is the struct type that holds all information for interacting with the filesystem
type OS struct {
	p []byte // buffer for memory optimized read
}

// NewOS creates a new Os and applies provided options from opts
func NewOS() *OS {
	// create object
	os := &OS{p: make([]byte, 32*1024)}
	return os
}

// CreateDir creates a new directory at name with the provided permissions. If a directory along the path
// does not exist, it is created as well.
func (o *OS) CreateDir(name string, perm fs.FileMode) error {

	if err := os.MkdirAll(name, perm); err != nil {
		return fmt.Errorf("failed to create directory (%s)", err)
	}

	return nil
}

// CreateFile creates a new file at name with the content from reader. If maxSize is set to a
// positive value, the file will be truncated to that size and en error thrown. If overwrite
// is set to true, the file will be overwritten if it already exists. If maxSize is set to -1,
// the file will be fully written. If a directory along the path does not exist, an error is thrown.
func (o *OS) CreateFile(name string, reader io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {

	// Check for file existence//overwrite
	if _, err := os.Lstat(name); !os.IsNotExist(err) {
		if !overwrite {
			return 0, fmt.Errorf("file already exists")
		}
	}

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
		return io.CopyBuffer(limitedWriter, reader, o.p)
	}

	// write data straight to file
	return io.CopyBuffer(dstFile, reader, o.p)
}

// CreateSymlink creates a new symlink at name pointing to target. If overwrite is set to true,
// the existing file will be overwritten. if name is a non-empty folder and overwrite is set, an error is thrown.
// If a folder along the path of name is missing, an error is thrown.
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
