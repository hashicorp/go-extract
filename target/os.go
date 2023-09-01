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
func checkForSymlinkInPath(path string) error {

	// ignore empty checks
	if len(path) == 0 {
		return nil
	}

	// dont check cwd
	if path == "." {
		return nil
	}

	// iterate over all directories and check for symlink
	allDirs := strings.Split(path, string(os.PathSeparator))
	for i := 0; i < len(allDirs); i++ {

		// build path
		checkPath := strings.Join(allDirs[0:i+1], string(os.PathSeparator))

		// perform check
		if stat, err := os.Lstat(strings.TrimSuffix(checkPath, string(os.PathSeparator))); !os.IsNotExist(err) {
			if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
				return fmt.Errorf("symlink in path")
			}
		}
	}

	// no symlink found within path
	return nil
}

// CreateSafeDir creates newDir in dstBase and checks for path traversal in directory name
func (o *Os) CreateSafeDir(config *config.Config, dstBase string, newDir string) error {

	// switch to destination
	oldLocation, err := os.Getwd()
	if err != nil {
		oldLocation = ""
	}
	defer os.Chdir(oldLocation)

	// go to extraction destination
	if err := os.Chdir(dstBase); err != nil {
		return err
	}

	// clean the directories
	newDir = filepath.Clean(newDir)

	// check that the new directory is within base
	if strings.HasPrefix(newDir, "..") {
		return fmt.Errorf("path traversal detected")
	}

	// check if directory already exist, then skip
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		return nil
	}

	// check if base directory is a symlink
	if err := checkForSymlinkInPath(filepath.Base(newDir)); err != nil {
		return err
	}

	// create dirs
	if err := os.MkdirAll(newDir, os.ModePerm); err != nil {
		return err
	}

	// output created dir
	log.Printf("+ %s%s", newDir, string(os.PathSeparator))

	return nil
}

// CreateSafeFile creates newFileName in dstBase with content from reader and file
// headers as provided in mode
func (o *Os) CreateSafeFile(config *config.Config, dstBase string, newFileName string, reader io.Reader, mode fs.FileMode) error {

	// switch to destination
	oldLocation, err := os.Getwd()
	if err != nil {
		oldLocation = ""
	}
	defer os.Chdir(oldLocation)

	// go to extraction destination
	if err := os.Chdir(dstBase); err != nil {
		return err
	}

	// check if a name is provided
	if len(newFileName) == 0 {
		return fmt.Errorf("cannot create file without name")
	}

	// clean filename
	newFileName = filepath.Clean(newFileName)

	// create target dir && check for path traversal
	if err := o.CreateSafeDir(config, ".", filepath.Dir(newFileName)); err != nil {
		return err
	}

	// check if base directory is a symlink
	if err := checkForSymlinkInPath(filepath.Dir(newFileName)); err != nil {
		return err
	}

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(newFileName); err == nil {
		if !config.Force {
			return fmt.Errorf("file already exists!")
		}
	}

	// create dst file
	dstFile, err := os.OpenFile(newFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() {
		dstFile.Close()
	}()

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

	_, err = io.Copy(dstFile, &bytesBuffer)
	if err != nil {
		return err
	}

	// output created file
	log.Printf("+ %s", newFileName)

	return nil
}

// CreateSymlink creates in dstBase a symlink newLinkName with destination linkTarget
func (o *Os) CreateSafeSymlink(config *config.Config, dstBase string, newLinkName string, linkTarget string) error {

	// switch to destination
	oldLocation, err := os.Getwd()
	if err != nil {
		oldLocation = ""
	}
	defer os.Chdir(oldLocation)

	// go to extraction destination
	if err := os.Chdir(dstBase); err != nil {
		return err
	}

	// check if symlink extraction is denied
	if config.DenySymlinks {
		log.Printf("skipped symlink extraction: %s -> %s", newLinkName, linkTarget)
		return nil
	}

	// check if a name is provided
	if len(newLinkName) == 0 {
		return fmt.Errorf("cannot create symlink without name")
	}

	// check absolut path for link target on unix
	if strings.HasPrefix(linkTarget, "/") {
		return fmt.Errorf("absolut path in symlink!")
	}

	// check absolut path for link target on windows
	if p := []rune(linkTarget); len(p) > 2 && p[1] == rune(':') {
		return fmt.Errorf("absolut path in symlink!")
	}

	// clean filename
	newLinkName = filepath.Clean(newLinkName)

	// check link target for traversal
	linkTargetCleaned := filepath.Join(filepath.Dir(newLinkName), linkTarget)
	if strings.HasPrefix(linkTargetCleaned, "..") {
		return fmt.Errorf("symlink path traversal detected!")
	}

	// create target dir && check for traversal in file name
	if err := o.CreateSafeDir(config, ".", filepath.Dir(newLinkName)); err != nil {
		return err
	}

	// check if base directory is a symlink
	if err := checkForSymlinkInPath(filepath.Dir(newLinkName)); err != nil {
		return err
	}

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(newLinkName); !os.IsNotExist(err) {
		if !config.Force {
			return fmt.Errorf("file already exist!")
		}

		// delete existing link
		if err := os.Remove(newLinkName); err != nil {
			return err
		}
	}

	// create link
	if err := os.Symlink(linkTarget, newLinkName); err != nil {
		return err
	}

	// output created link
	log.Printf("+ %s -> %s", newLinkName, linkTarget)

	return nil
}

// CreateTmpDir creates a temporary directory and returns its path
func CreateTmpDir() string {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		panic(err)
	}
	return tmpDir
}
