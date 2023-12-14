package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-extract/config"
)

// Os is the struct type that holds all information for interacting with the filesystem
type Os struct {
}

// NewOs creates a new Os and applies provided options from opts
func NewOs() *Os {

	// create object
	os := &Os{}

	return os
}

func securityCheckPath(config *config.Config, dstBase string, targetDirectory string) error {

	// clean the target
	targetDirectory = filepath.Clean(targetDirectory)

	// check for escape out of dstBase
	if !filepath.IsLocal(targetDirectory) {
		return fmt.Errorf("path traversal detected (%s)", targetDirectory)
	}

	// check each dir in path
	targetPathElements := strings.Split(targetDirectory, "/")
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
			if config.FollowSymlinks {
				config.Log.Debug("warning: following symlink", "sub-dir", subDirs)
			} else {
				return fmt.Errorf(fmt.Sprintf("symlink in path (%s) %s", subDirs, checkDir))
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

// CreateSafeDir creates newDir in dstBase and checks for path traversal in directory name
func (o *Os) CreateSafeDir(config *config.Config, dstBase string, newDir string) error {

	// check if dst exist
	if len(dstBase) > 0 {
		if _, err := os.Stat(dstBase); os.IsNotExist(err) {
			return fmt.Errorf("destination does not exist (%s)", dstBase)
		}
	}

	// no action needed
	if newDir == "." {
		return nil
	}

	// Check if newDir starts with an absolute path, if so -> remove
	if start := GetStartOfAbsolutePath(newDir); len(start) > 0 {
		config.Log.Debug("remove absolute path prefix", "prefix", start)
		newDir = strings.TrimPrefix(newDir, start)
	}

	if err := securityCheckPath(config, dstBase, newDir); err != nil {
		return err
	}

	// create dirs
	finalDirectoryPath := filepath.Join(dstBase, newDir)
	if err := os.MkdirAll(finalDirectoryPath, os.ModePerm); err != nil {
		return err
	}

	return nil
}

// CreateSafeFile creates newFileName in dstBase with content from reader and file
// headers as provided in mode
func (o *Os) CreateSafeFile(config *config.Config, dstBase string, newFileName string, reader io.Reader, mode fs.FileMode) error {

	// check if a name is provided
	if len(newFileName) == 0 {
		return fmt.Errorf("cannot create file without name")
	}

	// Check if newFileName starts with an absolute path, if so -> remove
	if start := GetStartOfAbsolutePath(newFileName); len(start) > 0 {
		config.Log.Debug("remove absolute path prefix", "prefix", start)
		newFileName = strings.TrimPrefix(newFileName, start)
	}

	// create target dir && check for path traversal // zip-slip
	if err := o.CreateSafeDir(config, dstBase, filepath.Dir(newFileName)); err != nil {
		return err
	}

	// Check for file existence//overwrite
	targetFile := filepath.Join(dstBase, newFileName)
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		if !config.Overwrite {
			return fmt.Errorf("file already exists (%s)", newFileName)
		}
	}

	// finally copy the data
	var sumRead int64
	p := make([]byte, 1024)
	var bytesBuffer bytes.Buffer

	for {
		n, err := reader.Read(p)
		if err != nil && err != io.EOF {
			return err
		}

		// nothing left to read, finished
		if n == 0 {
			break
		}

		// filesize check
		if err := config.CheckExtractionSize(sumRead + int64(n)); err != nil {
			return err
		}

		// store in buffer
		bytesBuffer.Write(p[:n])
		sumRead = sumRead + int64(n)
	}

	// create dst file
	dstFile, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() {
		dstFile.Close()
	}()

	// write data
	_, err = io.Copy(dstFile, &bytesBuffer)
	if err != nil {
		return err
	}

	return nil
}

// CreateSymlink creates in dstBase a symlink newLinkName with destination linkTarget
func (o *Os) CreateSafeSymlink(config *config.Config, dstBase string, newLinkName string, linkTarget string) error {

	// check if symlink extraction is denied
	if config.DenySymlinks {
		config.Log.Debug("skipped symlink extraction", newLinkName, linkTarget)
		return nil
	}

	// check if a name is provided
	if len(newLinkName) == 0 {
		return fmt.Errorf("cannot create symlink without name")
	}

	// Check if link target is absolute path
	if start := GetStartOfAbsolutePath(linkTarget); len(start) > 0 {

		// continue on error?
		if config.ContinueOnError {
			config.Log.Debug("skip link target with absolute path", "link target", linkTarget)
			return nil
		}

		// return error
		return fmt.Errorf("symlink with absolute path as target (%s)", linkTarget)
	}

	// clean filename
	newLinkName = filepath.Clean(newLinkName)
	newLinkDirectory := filepath.Dir(newLinkName)

	// create target dir && check for traversal in file name
	if err := o.CreateSafeDir(config, dstBase, newLinkDirectory); err != nil {
		return err
	}

	// check link target for traversal
	linkTargetCleaned := filepath.Join(newLinkDirectory, linkTarget)
	if err := securityCheckPath(config, dstBase, linkTargetCleaned); err != nil {
		return err
	}

	targetFile := filepath.Join(dstBase, newLinkName)

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		if !config.Overwrite {
			return fmt.Errorf("symlink already exist (%s)", newLinkName)
		}

		// delete existing link
		config.Log.Debug("overwrite symlink", "name", newLinkName)
		if err := os.Remove(targetFile); err != nil {
			return err
		}
	}

	// create link
	if err := os.Symlink(linkTarget, targetFile); err != nil {
		return err
	}

	return nil
}

// CreateTmpDir creates a temporary directory and returns its path
func CreateTmpDir() string {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		log.Printf("%s", err)
	}
	return tmpDir
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
