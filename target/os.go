package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-extract/config"
)

type Os struct {
	config *config.Config
}

// NewOs creates a new Os and applies provided options from opts
func NewOs(opts ...TargetOption) *Os {
	// defaults
	config := config.NewConfig()

	// create object
	os := &Os{
		config: config,
	}

	for _, opt := range opts {
		opt(os)
	}

	return os
}

// TODO(jan): on point functiion parameter and documentation

// CreateSafeDir creates newDir in dstBase and checks for path traversal in directory name
func (o *Os) CreateSafeDir(dstBase string, newDir string) error {

	// clean the directories
	dstBase = filepath.Clean(dstBase)
	newDir = filepath.Clean(newDir)

	// check that the new directory is within base
	if strings.HasPrefix(newDir, "..") {
		return fmt.Errorf("path traversal detected")
	}

	// compose new directory
	createDir := filepath.Clean(filepath.Join(dstBase, newDir))

	// create dirs
	if err := os.MkdirAll(createDir, os.ModePerm); err != nil {
		return err
	}

	return nil
}

// CreateSafeFile creates name in dstDir with conte nt from reader and file
// headers as provided in mode
func (o *Os) CreateSafeFile(dstDir string, name string, reader io.Reader, mode fs.FileMode) error {

	// create target dir && check for path traversal
	if err := o.CreateSafeDir(dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// target file will contain the content
	targetFile := filepath.Clean(filepath.Join(dstDir, name))

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFile); err == nil {
		if !o.config.Force {
			return fmt.Errorf("file already exists!")
		}
	}

	// create dst file
	dstFile, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
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
		if err := o.config.CheckExtractionSize(sumRead + int64(n)); err != nil {
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

	return nil
}

// CreateSymlink creates in dstDir a symlink name with destination linkTarget
func (o *Os) CreateSafeSymlink(dstDir string, name string, linkTarget string) error {

	// check absolut path for link target
	// TODO(jan): check for windows
	// TODO(jan): network drives concideration on win `\\<remote>`
	if strings.HasPrefix(linkTarget, "/") {
		return fmt.Errorf("absolut path in symlink!")
	}

	// check link target for traversal
	linkTargetCleaned := filepath.Clean(filepath.Join(filepath.Dir(name), linkTarget))
	if strings.HasPrefix(linkTargetCleaned, "..") {
		return fmt.Errorf("symlink path traversal detected!")
	}

	// create target dir && check for traversal in file name
	if err := o.CreateSafeDir(dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(filepath.Join(dstDir, name)); err == nil {
		if !o.config.Force {
			return fmt.Errorf("file already exist!")
		}
	}

	// create link
	if err := os.Symlink(linkTarget, filepath.Join(dstDir, name)); err != nil {
		return err
	}

	return nil
}

func (o *Os) SetConfig(config *config.Config) {
	o.config = config
}

func CreateTmpDir() string {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		panic(err)
	}
	return tmpDir
}
