// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {
	// CreateFile creates a file at the specified path with src as content. The mode parameter is the file mode that
	// should be set on the file. If the file already exists and overwrite is false, an error should be returned. If the
	// file does not exist, it should be created. The size of the file should not exceed maxSize. If the file is created
	// successfully, the number of bytes written should be returned. If an error occurs, the number of bytes written
	// should be returned along with the error. If maxSize < 0, the file size is not limited.
	CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error)

	// CreateDir creates at the specified path with the specified mode. If the directory already exists, nothing is done.
	// The function returns an error if there's a problem creating the directory. If the function completes successfully,
	// it returns nil.
	CreateDir(path string, mode fs.FileMode) error

	// CreateSymlink creates a symbolic link from newname to oldname. If newname already exists and overwrite is false,
	// the function returns an error. If newname already exists and overwrite is true, the function may overwrite the
	// existing symlink.
	CreateSymlink(oldname string, newname string, overwrite bool) error

	// Lstat see docs for os.Lstat. Main purpose is to check for symlinks in the extraction path
	// and for zip-slip attacks.
	Lstat(path string) (fs.FileInfo, error)

	// Stat see docs for os.Stat. Main purpose is to check if a symlink is pointing to a file or directory.
	Stat(path string) (fs.FileInfo, error)

	// Chmod see docs for os.Chmod. Main purpose is to set the file mode of a file or directory.
	Chmod(name string, mode fs.FileMode) error

	// Chtimes see docs for os.Chtimes. Main purpose is to set the file times of a file or directory.
	Chtimes(name string, atime, mtime time.Time) error

	// Lchtimes see docs for os.Lchtimes. Main purpose is to set the file times of a file or directory.
	Lchtimes(name string, atime, mtime time.Time) error

	// Chown see docs for os.Chown. Main purpose is to set the file owner and group of a file or directory.
	Chown(name string, uid, gid int) error
}

// createFile is a wrapper around the CreateFile function
//
// If the name is empty, the function returns an error.
//
// If the directory for the file does not exist, it will be created with the config.CustomCreateDirMode().
//
// If the path contains path traversal or a symlink, the function returns an error.
//
// If the path contains a symlink and config.TraverseSymlinks() returns false, the function returns an error.
//
// If the path contains a symlink and config.TraverseSymlinks() returns true, a warning is logged and the
// function continues.
//
// If the file is created successfully, the function returns the number of bytes written and nil.
func createFile(t Target, dst string, name string, src io.Reader, mode fs.FileMode, maxSize int64, cfg *Config) (int64, error) {
	// check if a name is provided
	if len(name) == 0 {
		return 0, fmt.Errorf("cannot create file without name")
	}

	// adjust path to by os specific
	parts := strings.Split(name, "/")
	name = filepath.Join(parts...)

	// check for traversal in file name, ensure the directory exist and is safe to write to.
	// If the directory does not exist, it will be created with the config.CustomCreateDirMode().
	fDir := filepath.Dir(name)

	// ensures that the directory exists and is safe to write to (e.g. no symlinks if disabled)
	if err := createDir(t, dst, fDir, cfg.CustomCreateDirMode(), cfg); err != nil {
		return 0, fmt.Errorf("cannot create directory: %w", err)
	}

	// ensure that if the file exist that it is not a symlink
	if err := securityCheck(t, dst, name, cfg); err != nil {
		return 0, fmt.Errorf("security check path failed: %w", err)
	}
	path := filepath.Join(dst, name)
	return t.CreateFile(path, src, mode, cfg.Overwrite(), maxSize)
}

// createDir is a wrapper around the CreateDir function
//
// If the name is empty, the function returns an error.
//
// If the directory for the symlink does not exist, it will be created with
// the config.CustomCreateDirMode().
//
// If the path contains path traversal or a symlink, the function returns an error.
//
// If the path contains a symlink and config.TraverseSymlinks() returns false, the function returns an error.
//
// If the path contains a symlink and config.TraverseSymlinks() returns true, a warning is logged and the
// function continues.
//
// If the directory is created successfully, the function returns nil.
func createDir(t Target, dst string, name string, mode fs.FileMode, cfg *Config) error {
	// check if dst exists
	if len(dst) > 0 {
		if _, err := t.Lstat(dst); os.IsNotExist(err) {
			if cfg.CreateDestination() {
				if err := t.CreateDir(dst, cfg.CustomCreateDirMode()); err != nil {
					return fmt.Errorf("failed to create destination directory %w", err)
				}
				cfg.Logger().Info("created destination directory", "path", dst)
			} else {
				return fmt.Errorf("destination does not exist")
			}
		}
	}

	// no action needed
	if name == "." {
		return nil
	}

	// perform security check to ensure that the path is safe to write to
	if err := securityCheck(t, dst, name, cfg); err != nil {
		return fmt.Errorf("security check path failed: %w", err)
	}

	// combine the path
	parts := strings.Split(name, "/")
	path := filepath.Join(dst, filepath.Join(parts...))
	return t.CreateDir(path, mode)
}

// createSymlink is a wrapper around the CreateSymlink function
//
// It checks if the symlink extraction is allowed and if the link target is an absolute path.
// If the symlink extraction is denied, the function returns an error. If the link target is an
// absolute path, the function returns an error.
//
// If the name is empty, the function returns an error .
//
// If the directory for the symlink does not exist, it will be created with the config.CustomCreateDirMode().
//
// If the path contains path traversal or a symlink, the function returns an error.
//
// If the path contains a symlink and config.TraverseSymlinks() returns false, the function returns an error.
//
// If the path contains a symlink and config.TraverseSymlinks() returns true, a warning is logged and the
// function continues.
//
// If the symlink is created successfully, the function returns nil.
func createSymlink(t Target, dst string, name string, linkTarget string, cfg *Config) error {
	// check if symlink extraction is denied
	if cfg.DenySymlinkExtraction() {
		return unsupportedFile(name)
	}

	// check if a name is provided
	if len(name) == 0 {
		return fmt.Errorf("empty name")
	}

	// Check if link target is absolute path
	if filepath.IsAbs(linkTarget) {

		// continue on error?
		if cfg.ContinueOnError() {
			cfg.Logger().Info("skip link target with absolute path", "link target", linkTarget)
			return nil
		}

		// return error
		return fmt.Errorf("symlink with absolute path as target: %s", linkTarget)
	}

	// convert name to platform specific path
	parts := strings.Split(name, "/")
	name = filepath.Join(parts...)

	// get link directory
	linkDirectory := filepath.Dir(name)

	// create target dir && check for traversal in file name
	if err := createDir(t, dst, linkDirectory, cfg.CustomCreateDirMode(), cfg); err != nil {

		if cfg.ContinueOnError() {
			cfg.Logger().Info("skip dir creation with error", "err", err)
			return nil
		}

		return fmt.Errorf("cannot create directory (%s) for symlink: %w", fmt.Sprintf("%s%s", linkDirectory, string(os.PathSeparator)), err)
	}

	// check link target for traversal
	targetCleaned := filepath.Join(linkDirectory, linkTarget)
	if err := securityCheck(t, dst, targetCleaned, cfg); err != nil {
		return fmt.Errorf("symlink target security check path failed: %w", err)
	}

	// create symlink
	return t.CreateSymlink(linkTarget, filepath.Join(dst, name), cfg.Overwrite())
}

// securityCheck checks if the targetDirectory contains path traversal
// and if the path contains a symlink.
//
// The function returns an error if the path contains path traversal or
// if a symlink is detected.
//
// If the path contains a symlink and config.TraverseSymlinks() returns true,
// a warning is logged and the function continues.
//
// If the path contains a symlink and config.TraverseSymlinks() returns false,
// an error is returned.
func securityCheck(t Target, dst string, path string, config *Config) error {
	// check if dstBase is empty, then targetDirectory should not be an absolute path
	if len(dst) == 0 {
		if filepath.IsAbs(path) {
			return fmt.Errorf("absolute path detected")
		}
	}

	// clean the target
	parts := strings.Split(path, "/")
	path = filepath.Join(parts...)

	// get relative path from base to new directory target
	rel, err := filepath.Rel(dst, filepath.Join(dst, path))
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}
	// check if the relative path is local
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("path traversal detected")
	}

	// check each dir in path
	targetPathElements := strings.Split(path, string(os.PathSeparator))
	for i := 0; i < len(targetPathElements); i++ {

		// assemble path
		subDirs := filepath.Join(targetPathElements[0 : i+1]...)
		checkDir := filepath.Join(dst, subDirs)

		// check if its a proper path
		if len(checkDir) == 0 {
			continue
		}

		if checkDir == "." {
			continue
		}

		// perform check if its a proper dir
		if _, err := t.Lstat(checkDir); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("invalid path: %w", err)
			}
		}

		// check for symlink
		isSymlink, err := isSymlink(t, checkDir)
		if err != nil {
			return fmt.Errorf("failed to check symlink: %w", err)
		}
		if isSymlink {
			if config.TraverseSymlinks() {
				config.Logger().Warn("traverse symlink", "sub-dir", subDirs)
			} else {
				return fmt.Errorf("symlink in path")
			}
		}
	}

	return nil
}

// isSymlink checks if path contains a symlink
//
// The function returns true if the path contains a symlink, otherwise false.
func isSymlink(t Target, path string) (bool, error) {
	// ignore empty checks
	if len(path) == 0 {
		return false, fmt.Errorf("empty path")
	}

	// don't check cwd
	if path == "." {
		return false, fmt.Errorf("cwd")
	}

	// perform check
	if stat, err := t.Lstat(path); !os.IsNotExist(err) {
		// check if error occurred --> not a symlink
		if err != nil {
			return false, fmt.Errorf("failed to check path: %w", err)
		}

		// check if we got stats
		if stat == nil {
			return false, fmt.Errorf("failed to get stats")
		}

		// check if symlink
		if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
			return true, nil
		}
	}

	// no symlink found within path
	return false, nil
}
