package extract

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Extract(ctx context.Context, src, dst string) error {

	// Extractors
	var unzip Zip
	var untar Tar

	// TODO(jan): determine correct extractor

	// create tmp directory
	tmpDir, err := os.MkdirTemp(os.TempDir(), "extract*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	tmpDir = filepath.Clean(tmpDir) + string(os.PathSeparator)

	// TODO(jan): add timeout
	// TODO(jan): detect filetype based on magic bytes
	switch {
	case strings.HasSuffix(src, ".zip"):
		// extract zip
		if err := unzip.Extract(src, tmpDir); err != nil {
			return err
		}
	case strings.HasSuffix(src, ".tar"):
		if err := untar.Extract(src, tmpDir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unspported archive type")

	}

	// move content from tmpDir to destination
	if err := CopyDirectory(tmpDir, dst); err != nil {
		return err
	}

	return nil
}

func createDir(dstDir, dirName string) error {

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

func createSymlink(dstDir string, name string, linkTarget string) error {

	// create target dir
	if err := createDir(dstDir, filepath.Dir(name)); err != nil {
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

func createFile(dstDir string, name string, reader io.Reader, mode fs.FileMode) error {

	// create target dir
	if err := createDir(dstDir, filepath.Dir(name)); err != nil {
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

	// TODO(jan): filesize check
	if _, err := io.Copy(dstFile, reader); err != nil {
		return err
	}

	return nil
}
