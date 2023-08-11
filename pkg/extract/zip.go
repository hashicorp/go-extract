package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	// TODO(jan): add timeout

	// walk over archive
	for _, f := range archive.File {
		dstFilePath := filepath.Clean(filepath.Join(dst, f.Name))

		// path sanitization
		if err := verifyPathPrefix(dst, dstFilePath); err != nil {
			return err
		}

		// handle directory
		if f.FileInfo().IsDir() {
			os.MkdirAll(dstFilePath, os.ModePerm)
			continue
		}

		// create sub dirs for file
		if err := os.MkdirAll(filepath.Dir(dstFilePath), os.ModePerm); err != nil {
			return err
		}

		// create dst file
		dstFile, err := os.OpenFile(dstFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		// open file in archive
		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		// TODO(jan): filesize check
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return err
		}

		// symlink check
		linkInfo, err := os.Lstat(dstFilePath)
		if linkInfo.Mode()&os.ModeSymlink == os.ModeSymlink {

			// check if file is a symlink
			origSymlinkDst, err := os.Readlink(dstFilePath)
			if err != nil {
				return err
			}
			expandedSymlinkDst := filepath.Clean(filepath.Join(dst, origSymlinkDst))
			if err := verifyPathPrefix(dst, expandedSymlinkDst); err != nil {
				return fmt.Errorf("symlink outside archive identified")
			}
		}

		dstFile.Close()
		fileInArchive.Close()
	}

	return nil
}
