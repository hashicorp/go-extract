package extractor

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// unpackTarget is the target that is used for extraction
var unpackTarget target.Target

// init initializes the unpackTarget
func init() {
	unpackTarget = target.NewOS()
}

// SetTarget sets the target for extraction
func SetTarget(t target.Target) {
	unpackTarget = t
}

// createFile is a wrapper around the target.CreateFile function
func createFile(config *config.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode, maxSize int64) (int64, error) {

	// check if a name is provided
	if len(name) == 0 {
		return 0, fmt.Errorf("cannot create file without name")
	}

	// check for traversal in file name, ensure the directory exist and is safe to write to.
	// If the directory does not exist, it will be created with the config.CustomCreateDirMode().
	fDir := filepath.Dir(name)
	if err := createDir(config, dstDir, fDir, config.CustomCreateDirMode()); err != nil {
		return 0, fmt.Errorf("cannot create directory (%s): %s", fDir, err)
	}

	return unpackTarget.CreateFile(dstDir, name, reader, mode, config.Overwrite(), maxSize)
}

// createDir is a wrapper around the target.CreateDir function
func createDir(config *config.Config, dstDir string, dirName string, mode fs.FileMode) error {

	// check if dst exist
	if len(dstDir) > 0 {
		if _, err := unpackTarget.Lstat(dstDir); os.IsNotExist(err) {
			if config.CreateDestination() {
				if err := unpackTarget.CreateDir(dstDir, ".", config.CustomCreateDirMode()); err != nil {
					return fmt.Errorf("failed to create destination directory %s", err)
				}
				config.Logger().Info("created destination directory", "path", dstDir)
			} else {
				return fmt.Errorf("destination does not exist (%s)", dstDir)
			}
		}
	}

	// no action needed
	if dirName == "." {
		return nil
	}

	if err := securityCheckPath(config, dstDir, dirName); err != nil {
		return fmt.Errorf("security check path failed: %w", err)
	}

	return unpackTarget.CreateDir(dstDir, dirName, mode)
}

// createSymlink is a wrapper around the target.CreateSymlink function
func createSymlink(config *config.Config, dstDir string, name string, target string) error {

	// check if symlink extraction is denied
	if config.DenySymlinkExtraction() {
		config.Logger().Info("skipped symlink extraction", name, target)
		return nil
	}

	// check if a name is provided
	if len(name) == 0 {
		return fmt.Errorf("cannot create symlink without name")
	}

	// Check if link target is absolute path
	if filepath.IsAbs(target) {

		// continue on error?
		if config.ContinueOnError() {
			config.Logger().Info("skip link target with absolute path", "link target", target)
			return nil
		}

		// return error
		return fmt.Errorf("symlink with absolute path as target (%s)", target)
	}

	// clean filename
	name = filepath.Clean(name)
	linkDirectory := filepath.Dir(name)

	// create target dir && check for traversal in file name
	if err := createDir(config, dstDir, linkDirectory, config.CustomCreateDirMode()); err != nil {
		return fmt.Errorf("cannot create directory (%s) for symlink: %w", fmt.Sprintf("%s%s", linkDirectory, string(os.PathSeparator)), err)
	}

	// check link target for traversal
	targetCleaned := filepath.Join(linkDirectory, target)
	if err := securityCheckPath(config, dstDir, targetCleaned); err != nil {
		return fmt.Errorf("symlink target security check path failed (%s)", target)
	}

	// create symlink
	return unpackTarget.CreateSymlink(dstDir, name, target, config.Overwrite())
}

// securityCheckPath checks if the targetDirectory contains path traversal
// and if the path contains a symlink. The function returns an error if the
// path contains path traversal or if a symlink is detected. If the path
// contains a symlink and config.FollowSymlinks() returns true, a warning is
// logged and the function continues. If the path contains a symlink and
// config.FollowSymlinks() returns false, an error is returned.
func securityCheckPath(config *config.Config, dstBase string, targetDirectory string) error {

	// clean the target
	targetDirectory = filepath.Clean(targetDirectory)

	// check if dstBase is empty, then targetDirectory should not be an absolute path
	if len(dstBase) == 0 {
		if filepath.IsAbs(targetDirectory) {
			return fmt.Errorf("absolute path detected (%s)", targetDirectory)
		}
	}

	// get relative path from base to new directory target
	rel, err := filepath.Rel(dstBase, filepath.Join(dstBase, targetDirectory))
	if err != nil {
		return fmt.Errorf("failed to get relative path (%s)", err)
	}
	// check if the relative path is local
	if strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path traversal detected (%s)", targetDirectory)
	}

	// check each dir in path
	targetPathElements := strings.Split(targetDirectory, string(os.PathSeparator))
	for i := 0; i < len(targetPathElements); i++ {

		// assemble path
		subDirs := filepath.Join(targetPathElements[0 : i+1]...)
		checkDir := filepath.Join(dstBase, subDirs)

		// check if its a proper path
		if len(checkDir) == 0 {
			continue
		}

		if checkDir == "." {
			continue
		}

		// perform check if its a proper dir
		if _, err := unpackTarget.Lstat(checkDir); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("invalid path")
			}

			// get out of the loop, bc/ don't check paths
			// for symlinks that does not exist
			if os.IsNotExist(err) {
				break
			}
		}

		// check for symlink
		if isSymlink(checkDir) {
			if config.FollowSymlinks() {
				config.Logger().Warn("following symlink", "sub-dir", subDirs)
			} else {
				target, err := getSymlinkTarget(checkDir)
				if err != nil {
					return fmt.Errorf("symlink in path: %s -> (error: %w)", checkDir, err)
				} else {
					return fmt.Errorf(fmt.Sprintf("symlink in path: %s -> %s", checkDir, target))
				}
			}
		}
	}

	return nil
}

// checkForSymlinkInPath checks if path contains a symlink
func isSymlink(path string) bool {

	// ignore empty checks
	if len(path) == 0 {
		return false
	}

	// don't check cwd
	if path == "." {
		return false
	}

	// perform check
	if stat, err := unpackTarget.Lstat(path); !os.IsNotExist(err) {
		if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
			return true
		}
	}

	// no symlink found within path
	return false
}

// getSymlinkTarget returns the target of a symlink
func getSymlinkTarget(path string) (string, error) {

	// check if path is a symlink
	if !isSymlink(path) {
		return "", fmt.Errorf("not a symlink")
	}

	// get target
	target, err := unpackTarget.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink target (%s)", err)
	}

	return target, nil

}
