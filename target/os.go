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

type Os struct{}

// CreateSafeDir creates in dstDir all directories that are provided in dirName
func (o *Os) CreateSafeDir(config *config.Config, dstDir string, dirName string) error {

	// get absolut path
	tragetDir := filepath.Clean(filepath.Join(dstDir, dirName)) + string(os.PathSeparator)

	// check path
	if !strings.HasPrefix(tragetDir, dstDir) {
		return fmt.Errorf("filename path traversal detected: %v", dirName)
	}

	// create dirs
	if err := os.MkdirAll(tragetDir, os.ModePerm); err != nil {
		return err
	}

	return nil
}

// CreateSymlink creates in dstDir a symlink name with destination linkTarget
func (o *Os) CreateSafeSymlink(config *config.Config, dstDir string, name string, linkTarget string) error {

	// create target dir
	if err := o.CreateSafeDir(config, dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, name))

	// Check for file existence and if it should be overwritten
	if _, err := os.Lstat(targetFilePath); err == nil {
		if !config.Overwrite {
			return fmt.Errorf("%v already exists!\n", name)
		} else {
			fmt.Printf("file %v exist and is going to be overwritten\n", name)
		}
	} else {
		fmt.Printf("%v new created\n", targetFilePath)
	}

	// check absolut path
	// TODO(jan): check for windows
	// TODO(jan): network drives concideration on win `\\<remote>`
	if strings.HasPrefix(linkTarget, "/") {
		return fmt.Errorf("symlink absolut path detected: %v", linkTarget)
	}

	// check relative path
	canonicalTarget := filepath.Clean(filepath.Join(dstDir, linkTarget))
	if !strings.HasPrefix(canonicalTarget, dstDir) {
		return fmt.Errorf("symlink path traversal detected: %v", linkTarget)
	}

	// create link
	if err := os.Symlink(linkTarget, targetFilePath); err != nil {
		return err
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
	if _, err := os.Lstat(targetFilePath); err == nil && !config.Overwrite {
		return fmt.Errorf("%v already exists!", name)
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
	writtenBytes, err := io.Copy(dstFile, reader)
	if err != nil {
		return err
	}

	// check if too much bytes written
	if err := config.CheckFileSize(writtenBytes); err != nil {
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
