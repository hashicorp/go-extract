package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
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

	// TODO(jan): add timeout

	// walk over archive
	for _, f := range archive.File {
		dstFilePath := filepath.Clean(filepath.Join(dst, f.Name))

		// path sanitization
		if err := verifyPathPrefix(dst, dstFilePath); err != nil {
			return fmt.Errorf("%v: %v", err, f.Name)
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

		// check for symlink
		if f.FileHeader.Mode()&os.ModeType == os.ModeSymlink {

			// read content to determine symlink destination
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() {
				if err := rc.Close(); err != nil {
					panic(err)
				}
			}()
			data, err := io.ReadAll(rc)
			symlinkDst := string(data)
			if err != nil {
				return err
			}

			// check symlink destination
			if strings.HasPrefix(symlinkDst, "/") {
				return fmt.Errorf("symlink with absolut path: %v", symlinkDst)
			}
			canonicalTarget := filepath.Clean(filepath.Join(dst, symlinkDst))
			if err := verifyPathPrefix(dst, canonicalTarget); err != nil {
				return fmt.Errorf("%v: %v", err, symlinkDst)
			}

			writeSymbolicLink(dstFilePath, symlinkDst)
			continue
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
		log.Printf("copy from archive: %v, %v", dstFile.Name(), f.FileHeader.Name)
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

func writeSymbolicLink(filePath string, targetPath string) error {
	log.Printf("writeSymbolicLink(filePath, targetPath): %v, %v", filePath, targetPath)

	// create dirs
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	// create link
	if err := os.Symlink(targetPath, filePath); err != nil {
		return err
	}

	return nil
}
