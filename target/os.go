package target

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-extract"
)

type Os struct{}

// CreateSafeDir creates in dstDir all directories that are provided in dirName
func (o *Os) CreateSafeDir(dstDir string, dirName string) error {

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
func (o *Os) CreateSafeSymlink(dstDir string, name string, linkTarget string) error {

	// create target dir
	if err := o.CreateSafeDir(dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, name))

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
func (o *Os) CreateSafeFile(config *extract.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode) error {

	// create target dir
	if err := o.CreateSafeDir(dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, name))

	// create dst file
	dstFile, err := os.OpenFile(targetFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
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
