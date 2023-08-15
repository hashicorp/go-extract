package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// base reference: https://golang.cafe/blog/golang-unzip-file-example.html

type Zip struct {
}

func (z *Zip) Extract(src, dst string) error {

	// open archive
	archive, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer archive.Close()

	// walk over archive
	for _, archiveFile := range archive.File {

		switch archiveFile.FileHeader.Mode() & os.ModeType {
		case os.ModeDir:
			// handle directory
			if err := createDir(dst, archiveFile.Name); err != nil {
				return err
			}
			continue

		case os.ModeSymlink:
			// handle symlink
			if err := createSymlink(dst, archiveFile); err != nil {
				return err
			}
			continue

		default:
			// handle files
			if err := createFile(dst, archiveFile); err != nil {
				return err
			}
		}

	}

	return nil
}

func createDir(dstDir, dirName string) error {

	// get absolut path
	tragetDir := filepath.Clean(filepath.Join(dstDir, dirName)) + string(os.PathSeparator)

	// check path
	if !strings.HasPrefix(tragetDir, dstDir) {
		return fmt.Errorf("path traversal detected: %v", dirName)
	}

	// create dirs
	if err := os.MkdirAll(tragetDir, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func createSymlink(dstDir string, f *zip.File) error {

	// create target dir
	if err := createDir(dstDir, filepath.Dir(f.Name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, f.Name))

	// read content to determine symlink destination
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		rc.Close()
	}()
	data, err := io.ReadAll(rc)
	symlinkTarget := string(data)
	if err != nil {
		return err
	}

	// check absolut path
	// TODO(jan): check for windows
	// TODO(jan): network drives concideration on win `\\<remote>`
	if strings.HasPrefix(symlinkTarget, "/") {
		return fmt.Errorf("absolut path detected: %v", symlinkTarget)
	}

	// check relative path
	canonicalTarget := filepath.Clean(filepath.Join(dstDir, symlinkTarget))
	if !strings.HasPrefix(canonicalTarget, dstDir) {
		return fmt.Errorf("path traversal detected: %v", symlinkTarget)
	}

	// write the final link
	if err := writeSymbolicLink(targetFilePath, symlinkTarget); err != nil {
		return err
	}

	return nil
}

func createFile(dstDir string, f *zip.File) error {

	// create target dir
	if err := createDir(dstDir, filepath.Dir(f.Name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, f.Name))

	// create dst file
	dstFile, err := os.OpenFile(targetFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer func() {
		dstFile.Close()
	}()

	// open file in archive
	fileInArchive, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		fileInArchive.Close()
	}()

	// TODO(jan): filesize check
	if _, err := io.Copy(dstFile, fileInArchive); err != nil {
		return err
	}

	return nil
}
