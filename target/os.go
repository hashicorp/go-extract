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

type Os struct{}

func NewOs() Os {
	return Os{}
}

// TODO(jan): on point functiion parameter and documentation

// CreateSafeDir creates in dstDir all directories that are provided in dirName
func (o *Os) CreateSafeDir(config *config.Config, dstBase string, newDir string) error {

	// absolut path for destination
	dstBase, err := filepath.Abs(dstBase)
	if err != nil {
		return fmt.Errorf("CreateSafeDir::cannot get filepath.Abs(): %v", err)
	}
	dstBase = dstBase + string(os.PathSeparator)

	// get absolut path for new dir
	newDir = filepath.Clean(filepath.Join(dstBase, newDir)) + string(os.PathSeparator)

	// check that the new directory is within base
	if !strings.HasPrefix(newDir, dstBase) {
		return fmt.Errorf("filename path traversal detected: %v", newDir)
	}

	// create dirs
	if err := os.MkdirAll(newDir, os.ModePerm); err != nil {
		return fmt.Errorf("cannot create directories: %v", err)
	}

	return nil
}

// CreateSafeFile creates name in dstDir with conte nt from reader and file
// headers as provided in mode
func (o *Os) CreateSafeFile(config *config.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode) error {

	// create target dir
	if err := o.CreateSafeDir(config, dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, name))

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFilePath); err == nil {
		if !config.Force {
			return fmt.Errorf("already exists: %v", name)
		}
	}

	// create dst file
	dstFile, err := os.OpenFile(targetFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("openFile error: %v", err)
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

		// filesize check
		if err := config.CheckFileSize(sumRead + int64(n)); err != nil {
			return err
		}

		// nothing left to read, finished
		if n == 0 {
			break
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
func (o *Os) CreateSafeSymlink(config *config.Config, dstDir string, name string, linkTarget string) error {

	// get absolut path of destination
	dstDirAbsolut, err := filepath.Abs(dstDir)
	if err != nil {
		return err
	}
	dstDirAbsolut = dstDirAbsolut + string(os.PathSeparator)

	// absolut path and directory of new file
	targetFilePathAbsolut := filepath.Clean(filepath.Join(dstDirAbsolut, name))
	targetFileDirAbsolut := filepath.Dir(targetFilePathAbsolut)

	// create target dir // check for traversal in file name
	if err := o.CreateSafeDir(config, dstDirAbsolut, filepath.Dir(name)); err != nil {
		return err
	}

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFilePathAbsolut); err == nil {
		if !config.Force {
			return fmt.Errorf("file already exist!")
		}
	}

	// check absolut path for link target
	// TODO(jan): check for windows
	// TODO(jan): network drives concideration on win `\\<remote>`
	if strings.HasPrefix(linkTarget, "/") {
		return fmt.Errorf("absolut path in symlink!")
	}

	// expand link to get absolut path of target
	linkTargetAbsolut, err := filepath.Abs(filepath.Join(targetFileDirAbsolut, linkTarget))
	if err != nil {
		return err
	}

	if !strings.HasPrefix(linkTargetAbsolut, dstDirAbsolut) {
		return fmt.Errorf("symlink path traversal detected!")
	}

	// create link
	if err := os.Symlink(linkTarget, targetFilePathAbsolut); err != nil {
		return err
	}

	return nil
}

func CreateTmpDir() string {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		panic(err)
	}
	return tmpDir
}
