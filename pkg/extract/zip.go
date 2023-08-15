package extract

import (
	"archive/zip"
	"io"
	"log"
	"os"
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

		fileMode := archiveFile.FileHeader.Mode() & os.ModeType

		switch fileMode {
		case os.ModeDir:
			// handle directory
			if err := createDir(dst, archiveFile.Name); err != nil {
				return err
			}
			continue

		case os.ModeSymlink:
			// handle symlink
			linkTarget, err := readLinkTargetFromZip(archiveFile)
			if err != nil {
				return err
			}
			if err := createSymlink(dst, archiveFile.Name, linkTarget); err != nil {
				return err
			}
			continue

		// in case of a normal file the value is not set
		case 0:
			if err := createFileFromZip(dst, archiveFile); err != nil {
				return err
			}

		// catch all for unspported file modes
		default:
			log.Printf("unspported filemode: %s", fileMode)

		}

	}

	return nil
}

func readLinkTargetFromZip(f *zip.File) (string, error) {
	// read content to determine symlink destination
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer func() {
		rc.Close()
	}()
	data, err := io.ReadAll(rc)
	symlinkTarget := string(data)
	if err != nil {
		return "", err
	}

	return symlinkTarget, nil
}

func createFileFromZip(dstDir string, f *zip.File) error {

	// open file in archive
	fileInArchive, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		fileInArchive.Close()
	}()

	// create the file
	if err := createFile(dstDir, f.Name, fileInArchive, f.Mode()); err != nil {
		return err
	}

	return nil
}
