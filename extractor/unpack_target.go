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

// createFile creates a new file at name with the content from reader. If maxSize is set to a
// positive value, the file will be truncated to that size and en error thrown. If overwrite
// is set to true, the file will be overwritten if it already exists. If maxSize is set to -1,
// the file will be fully written. If a directory along the path does not exist, an error is thrown.
func createFile(c *config.Config, base string, name string, reader io.Reader, perm fs.FileMode, maxSize int64) (int64, error) {

	// check if a name is provided
	if len(name) == 0 {
		return 0, fmt.Errorf("cannot create file without name")
	}

	// Check if newFileName starts with an absolute path, if so -> remove
	if start := getStartOfAbsolutePath(name); len(start) > 0 {
		c.Logger().Debug("remove absolute path prefix", "prefix", start)
		name = strings.TrimPrefix(name, start)
	}

	// check for path traversal // zip-slip
	newFilePath := filepath.Dir(name)
	if err := securityCheckPath(c, base, newFilePath); err != nil {
		return 0, fmt.Errorf("security check failed: %w", err)
	}

	// create file
	newFilePath = filepath.Join(base, name)
	return unpackTarget.CreateFile(newFilePath, reader, perm, c.Overwrite(), maxSize)
}

// createDir creates a new directory at name with the provided permissions. If a directory along the base
// does not exist, it is created as well. If the directory already exists, the creation it is skipped.
func createDir(config *config.Config, base string, name string, perm fs.FileMode) error {
	// Check if newDir starts with an absolute path, if so -> remove
	if start := getStartOfAbsolutePath(name); len(start) > 0 {
		config.Logger().Debug("remove absolute path prefix", "prefix", start)
		name = strings.TrimPrefix(name, start)
	}

	if err := securityCheckPath(config, base, name); err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}

	// create directory
	finalDirectoryPath := filepath.Join(base, name)
	if _, err := lstat(finalDirectoryPath); os.IsNotExist(err) {
		return unpackTarget.CreateDir(finalDirectoryPath, perm)
	}

	// directory already exists
	return nil
}

// createSymlink creates a new symlink at name pointing to target. If overwrite is set to true,
// the existing file will be overwritten. if name is a non-empty folder and overwrite is set, an error is thrown.
// If a folder along the path of name is missing, an error is thrown.
func createSymlink(config *config.Config, base string, name string, target string) error {

	// check if symlink extraction is denied
	if config.DenySymlinkExtraction() {
		config.Logger().Info("skipped symlink extraction (disabled)", "name", name, "target", target)
		return nil
	}

	// check if a name is provided
	if len(name) == 0 {
		return fmt.Errorf("cannot create symlink without name")
	}

	// clean filename
	name = filepath.Clean(name)
	path := filepath.Dir(name)

	// check for path traversal // zip-slip
	if err := securityCheckPath(config, base, path); err != nil {
		return fmt.Errorf("symlink name security check failed: %w", err)
	}

	// Check if link target is absolute path
	if start := getStartOfAbsolutePath(target); len(start) > 0 {
		return fmt.Errorf("symlink with absolute path as target: %s", target)
	}

	// check link target for traversal
	linkTargetCleaned := filepath.Join(path, target)
	if err := securityCheckPath(config, base, linkTargetCleaned); err != nil {
		return fmt.Errorf("symlink target security check failed: %w", err)
	}

	// create symlink
	linkName := filepath.Join(base, name)
	return unpackTarget.CreateSymlink(linkName, target, config.Overwrite())
}

// lstat is a wrapper for the target.Lstat function
func lstat(path string) (fs.FileInfo, error) {
	return unpackTarget.Lstat(path)
}

// SetTarget sets the target that is used for extraction
func SetTarget(t target.Target) {
	unpackTarget = t
}

// securityCheckPath checks if the targetDirectory contains a path traversal, an error is returned.
// The function checks if the targetDirectory contains a symlink. If symlink following is enabled,
// a warning is logged. If symlink following is disabled, an error is returned.
func securityCheckPath(config *config.Config, dstBase string, targetDirectory string) error {

	// clean the target
	targetDirectory = filepath.Clean(targetDirectory)

	// check for escape out of dstBase
	if !filepath.IsLocal(targetDirectory) {
		return fmt.Errorf("path traversal detected: %s", targetDirectory)
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
		if _, err := lstat(checkDir); err != nil {
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
	if stat, err := lstat(path); !os.IsNotExist(err) {
		if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
			return true
		}
	}

	// no symlink found within path
	return false
}

func getStartOfAbsolutePath(path string) string {

	// check absolute path for link target on unix
	if strings.HasPrefix(path, "/") {
		return fmt.Sprintf("%s%s", "/", getStartOfAbsolutePath(path[1:]))
	}

	// check absolute path for link target on unix
	if strings.HasPrefix(path, `\`) {
		return fmt.Sprintf("%s%s", `\`, getStartOfAbsolutePath(path[1:]))
	}

	// check absolute path for link target on windows
	if p := []rune(path); len(p) > 2 && p[1] == rune(':') {
		return fmt.Sprintf("%s%s", path[0:3], getStartOfAbsolutePath(path[3:]))
	}

	return ""
}

// getSymlinkTarget returns the target of a symlink
func getSymlinkTarget(path string) (string, error) {

	// check if path is a symlink
	if !isSymlink(path) {
		return "", fmt.Errorf("not a symlink")
	}

	// get target
	target, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink target (%s)", err)
	}

	return target, nil

}
