// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// TargetDisk is the struct type that holds all information for interacting with the filesystem
type TargetDisk struct{}

// NewTargetDisk creates a new Os and applies provided options from opts
func NewTargetDisk() *TargetDisk {
	// create object
	td := &TargetDisk{}
	return td
}

// CreateDir creates a directory at the specified path with the specified mode. If the directory already
// exists, nothing is done.
func (d *TargetDisk) CreateDir(path string, mode fs.FileMode) error {

	// create dirs
	if err := os.MkdirAll(path, mode.Perm()); err != nil {
		return fmt.Errorf("failed to create directory (%w)", err)
	}

	return nil
}

// CreateFile creates a file at the specified path with src as content.
// The mode parameter is the file mode that should be set on the file. If the file already exists and
// overwrite is false, an error should be returned. If the file does not exist, it should be created.
// The size of the file should not exceed maxSize. If the file is created successfully, the number of bytes written
// should be returned. If an error occurs, the number of bytes written should be returned along with the error.
// If maxSize < 0, the file size is not limited.
func (d *TargetDisk) CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {
	// Check for path validity and if file existence+overwrite
	if _, err := os.Lstat(path); !os.IsNotExist(err) {

		// something wrong with path
		if err != nil {
			return 0, fmt.Errorf("invalid path: %w", err)
		}

		// check for overwrite
		if !overwrite {
			return 0, fmt.Errorf("file already exists")
		}
	}

	// create dst file
	dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		dstFile.Close()
	}()

	// write data to file
	writer := limitWriter(dstFile, maxSize)
	n, err := io.Copy(writer, src)
	if err != nil {
		return n, fmt.Errorf("failed to write file: %w", err)
	}

	return n, err
}

// CreateSymlink creates a symbolic link from newname to oldname. If
// newname already exists and overwrite is false, an error should be returned.
func (d *TargetDisk) CreateSymlink(oldname string, newname string, overwrite bool) error {

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(newname); !os.IsNotExist(err) {
		if !overwrite {
			return fmt.Errorf("file already exist")
		}

		// delete existing link
		if err := os.Remove(newname); err != nil {
			return fmt.Errorf("failed to overwrite file: %w", err)
		}
	}

	// create link
	if err := os.Symlink(oldname, newname); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil

}

// Lstat returns the FileInfo structure describing the named file.
// If there is an error, it will be of type *PathError.
func (d *TargetDisk) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(name)
}

// Stat returns the FileInfo structure describing the named file.
// If there is an error, it will be of type *PathError.
func (d *TargetDisk) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// Chmod changes the mode of the named file to mode.
func (d *TargetDisk) Chmod(name string, mode fs.FileMode) error {
	return os.Chmod(name, mode.Perm())
}

// Chtimes changes the access and modification times of the named file.
func (d *TargetDisk) Chtimes(name string, atime, mtime time.Time) error {
	return os.Chtimes(name, atime, mtime)
}

// Chown changes the numeric uid and gid of the named file.
func (d *TargetDisk) Chown(name string, uid, gid int) error {
	if os.Geteuid() != 0 {
		return nil
	}
	return os.Lchown(name, uid, gid)
}

// Lchtimes changes the access and modification times of the named file.
func (d *TargetDisk) Lchtimes(name string, atime, mtime time.Time) error {
	if canMaintainSymlinkTimestamps {
		return lchtimes(name, atime, mtime)
	}
	return nil
}
