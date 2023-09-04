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

// checkForSymlinkInPath checks if path contains a symlink
func checkForSymlinkInPath(dstBase string, path string) error {

	// iterate over each sub-directory and check that
	dirs := strings.Split(path, string(os.PathSeparator))
	for i := 0; i < len(dirs); i++ {
		subDirs := filepath.Join(dirs[0 : i+1]...)
		if isSymlink(filepath.Join(dstBase, subDirs)) {
			return fmt.Errorf(fmt.Sprintf("symlink in path (%s)", subDirs))
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

	// dont check cwd
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

	// check if file starts with absolut path
	if start := GetStartOfAbsolutPath(newDir); len(start) > 0 {

		// continue on error?
		if config.ContinueOnError {
			config.Log.Printf("skip file with absolut path (%s)", newDir)
			return nil
		}

		// return error
		return fmt.Errorf("file with absolut path (%s)", newDir)
	}

	// clean the directories
	newDir = filepath.Clean(newDir)

	// check that the new directory is within base
	if strings.HasPrefix(newDir, "..") {
		return fmt.Errorf("path traversal detected (%s)", newDir)
	}

	// check if base directory is a symlink
	if err := checkForSymlinkInPath(dstBase, filepath.Dir(newDir)); err != nil {

		// allow following sym links
		if config.FollowSymlinks {
			config.Log.Printf("warning: following symlink (%s)", filepath.Dir(newDir))
		} else {
			return err
		}
	}

	finalDirectoryPath := filepath.Join(dstBase, newDir)

	// check if directory already exist, then skip
	if _, err := os.Stat(finalDirectoryPath); !os.IsNotExist(err) {
		return nil
	}

	// create dirs
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

	// check if file starts with absolut path
	if start := GetStartOfAbsolutPath(newFileName); len(start) > 0 {

		// continue on error?
		if config.ContinueOnError {
			config.Log.Printf("skip file with absolut path (%s)", newFileName)
			return nil
		}

		// return error
		return fmt.Errorf("file with absolut path (%s)", newFileName)
	}

	// clean filename
	newFileName = filepath.Clean(newFileName)

	// check if base directory is a symlink
	if err := checkForSymlinkInPath(dstBase, filepath.Dir(newFileName)); err != nil {
		// allow following sym links
		if config.FollowSymlinks {
			config.Log.Printf("warning: following symlink (%s)", filepath.Dir(newFileName))
		} else {
			return err
		}
	}

	// create target dir && check for path traversal
	if err := o.CreateSafeDir(config, dstBase, filepath.Dir(newFileName)); err != nil {
		return err
	}

	targetFile := filepath.Join(dstBase, newFileName)

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFile); err == nil {
		if !config.Overwrite {
			return fmt.Errorf("file already exists (%s)", newFileName)
		}
	}

	// finaly copy the data
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
		config.Log.Printf("skipped symlink extraction: %s -> %s", newLinkName, linkTarget)
		return nil
	}

	// check if a name is provided
	if len(newLinkName) == 0 {
		return fmt.Errorf("cannot create symlink without name")
	}

	// check if file starts with absolut path
	if start := GetStartOfAbsolutPath(newLinkName); len(start) > 0 {

		// continue on error?
		if config.ContinueOnError {
			config.Log.Printf("skip file with absolut path (%s)", newLinkName)
			return nil
		}

		// return error
		return fmt.Errorf("file with absolut path (%s)", newLinkName)
	}

	// Check if link target is absolut path
	if start := GetStartOfAbsolutPath(linkTarget); len(start) > 0 {

		// continue on error?
		if config.ContinueOnError {
			config.Log.Printf("skip link target with absolut path (%s)", linkTarget)
			return nil
		}

		// return error
		return fmt.Errorf("symlink with absolut path as target (%s)", linkTarget)
	}

	// clean filename
	newLinkName = filepath.Clean(newLinkName)
	newLinkDirectory := filepath.Dir(newLinkName)

	// check link target for traversal
	linkTargetCleaned := filepath.Join(newLinkDirectory, linkTarget)
	if strings.HasPrefix(linkTargetCleaned, "..") {
		return fmt.Errorf("symlink path traversal detected (%s)", linkTargetCleaned)
	}

	// check if base directory is a symlink
	if err := checkForSymlinkInPath(dstBase, newLinkDirectory); err != nil {
		// allow following sym links
		if config.FollowSymlinks {
			config.Log.Printf("Warning: following symlink (%s)", newLinkDirectory)
		} else {
			return err
		}
	}

	// create target dir && check for traversal in file name
	if err := o.CreateSafeDir(config, dstBase, newLinkDirectory); err != nil {
		return err
	}

	targetFile := filepath.Join(dstBase, newLinkName)

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		if !config.Overwrite {
			return fmt.Errorf("symlink already exist (%s)", newLinkName)
		}

		// delete existing link
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

func GetStartOfAbsolutPath(path string) string {

	// check absolut path for link target on unix
	if strings.HasPrefix(path, "/") {
		return "/"
	}

	// check absolut path for link target on windows
	if p := []rune(path); len(p) > 2 && p[1] == rune(':') {
		return path[0:3]
	}

	return ""
}
