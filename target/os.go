package target

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-extract/config"
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

// securityCheckPath checks if the targetDirectory contains path traversal
// and if the path contains a symlink. The function returns an error if the
// path contains path traversal or if a symlink is detected. If the path
// contains a symlink and config.FollowSymlinks() returns true, a warning is
// logged and the function continues. If the path contains a symlink and
// config.FollowSymlinks() returns false, an error is returned.
func securityCheckPath(config *config.Config, dstBase string, targetDirectory string) error {

	// clean the target
	targetDirectory = filepath.Clean(targetDirectory)

	// get relative path from base to new directory target
	rel, err := filepath.Rel(dstBase, filepath.Join(dstBase, targetDirectory))
	if err != nil {
		return fmt.Errorf("failed to get relative path (%s)", err)
	}
	// check if the relative path is local
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("path traversal detected (dstBase: %s, rel: %s): %s, joined: %s", dstBase, rel, targetDirectory, filepath.Join(dstBase, targetDirectory))
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
		if _, err := os.Lstat(checkDir); err != nil {
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
	if stat, err := os.Lstat(path); !os.IsNotExist(err) {
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
	target, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink target (%s)", err)
	}

	return target, nil

}

// CreateSafeDir creates newDir in dstBase and checks for path traversal in directory name.
// If dstBase is empty, the directory will be created in the current working directory. If dstBase
// does not exist and config.CreateDestination() returns true, it will be created with the
// config.CustomCreateDirMode(). The mode parameter is the file mode that should be set on the directory.
// If the directory already exists, the mode will be set on the directory.
func (o *OS) CreateSafeDir(config *config.Config, dstBase string, newDir string, mode fs.FileMode) error {

	// check if dst exist
	if len(dstBase) > 0 {
		if _, err := os.Stat(dstBase); os.IsNotExist(err) {
			if config.CreateDestination() {
				if err := os.MkdirAll(dstBase, config.CustomCreateDirMode().Perm()); err != nil {
					return fmt.Errorf("failed to create destination directory %s", err)
				}
				// ensure file permission is set regardless the umask
				if err := os.Chmod(dstBase, config.CustomCreateDirMode().Perm()); err != nil {
					return fmt.Errorf("failed to set folder permission (%s)", err)
				}
				config.Logger().Info("created destination directory", "path", dstBase)
			} else {
				return fmt.Errorf("destination does not exist (%s)", dstBase)
			}
		}
	}

	// no action needed
	if newDir == "." {
		return nil
	}

	if err := securityCheckPath(config, dstBase, newDir); err != nil {
		return fmt.Errorf("security check path failed: %w", err)
	}

	// create dirs
	finalDirectoryPath := filepath.Join(dstBase, newDir)
	if err := os.MkdirAll(finalDirectoryPath, mode.Perm()); err != nil {
		return fmt.Errorf("failed to create directory (%s)", err)
	}

	return nil
}

// CreateSafeFile creates newFileName in dstBase with content from reader and file
// headers as provided in mode. If dstBase is empty, the file will be created in the current
// working directory. If dstBase does not exist and config.CreateDestination() returns true, it will be created with the
// config.CustomCreateDirMode(). The mode parameter is the file mode that is set on the file.
// If the path of the file contains path traversal, an error should be returned. If the path *to the file* (not dstBase) does not
// exist, the directories is created with the config.CustomCreateDirMode() by the implementation.
func (o *OS) CreateSafeFile(cfg *config.Config, dstBase string, newFileName string, reader io.Reader, mode fs.FileMode) error {

	// check if a name is provided
	if len(newFileName) == 0 {
		return fmt.Errorf("cannot create file without name")
	}

	// check for traversal in file name, ensure the directory exist and is safe to write to.
	// If the directory does not exist, it will be created with the config.CustomCreateDirMode().
	if err := o.CreateSafeDir(cfg, dstBase, filepath.Dir(newFileName), cfg.CustomCreateDirMode()); err != nil {
		return fmt.Errorf("cannot create directory for file (%s)", err)
	}

	// Check for file existence//overwrite
	targetFile := filepath.Join(dstBase, newFileName)
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		if !cfg.Overwrite() {
			return fmt.Errorf("file already exists (%s)", newFileName)
		}
	}

	// create dst file
	dstFile, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("failed to create file (%s)", err)
	}
	defer func() {
		dstFile.Close()
	}()

	// check if a max extraction size is set
	if cfg.MaxExtractionSize() >= 0 {

		// encapsulate reader with limit reader
		limitedWriter := NewLimitErrorWriter(dstFile, cfg.MaxExtractionSize())
		if _, err = io.Copy(limitedWriter, reader); err != nil {
			return fmt.Errorf("failed to write file (%s)", err)
		}

	} else {

		// write data straight to file
		if _, err = io.Copy(dstFile, reader); err != nil {
			return fmt.Errorf("failed to write file (%s)", err)
		}
	}

	return nil
}

// CreateSymlink creates in dstBase a symlink newLinkName with destination linkTarget.
// If dstBase is empty, the symlink is created in the current working directory. If dstBase
// does not exist and config.CreateDestination() returns true, it will be created with the
// config.CustomCreateDirMode(). If the path of the symlink contains path traversal, an error
// is returned. If the path *to the symlink* (not dstBase) does not exist, the directories
// is created with the config.CustomCreateDirMode(). If the symlink already exists and
// config.Overwrite() returns false, an error is returned.
func (o *OS) CreateSafeSymlink(config *config.Config, dstBase string, newLinkName string, linkTarget string) error {

	// check if symlink extraction is denied
	if config.DenySymlinkExtraction() {
		config.Logger().Info("skipped symlink extraction", newLinkName, linkTarget)
		return nil
	}

	// check if a name is provided
	if len(newLinkName) == 0 {
		return fmt.Errorf("cannot create symlink without name")
	}

	// Check if link target is absolute path
	if filepath.IsAbs(linkTarget) {

		// continue on error?
		if config.ContinueOnError() {
			config.Logger().Info("skip link target with absolute path", "link target", linkTarget)
			return nil
		}

		// return error
		return fmt.Errorf("symlink with absolute path as target (%s)", linkTarget)
	}

	// clean filename
	newLinkName = filepath.Clean(newLinkName)
	newLinkDirectory := filepath.Dir(newLinkName)

	// create target dir && check for traversal in file name
	if err := o.CreateSafeDir(config, dstBase, newLinkDirectory, config.CustomCreateDirMode()); err != nil {
		return fmt.Errorf("cannot create directory (%s) for symlink: %w", fmt.Sprintf("%s%s", newLinkDirectory, string(os.PathSeparator)), err)
	}

	// check link target for traversal
	linkTargetCleaned := filepath.Join(newLinkDirectory, linkTarget)
	if err := securityCheckPath(config, dstBase, linkTargetCleaned); err != nil {
		return fmt.Errorf("symlink target security check path failed (%s)", linkTarget)
	}

	targetFile := filepath.Join(dstBase, newLinkName)

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		if !config.Overwrite() {
			return fmt.Errorf("symlink already exist (%s)", newLinkName)
		}

		// delete existing link
		config.Logger().Warn("overwrite symlink", "name", newLinkName)
		if err := os.Remove(targetFile); err != nil {
			return fmt.Errorf("failed to remove existing symlink (%s)", err)
		}
	}

	// create link
	if err := os.Symlink(linkTarget, targetFile); err != nil {
		return fmt.Errorf("failed to create symlink (%s)", err)
	}

	return nil
}
