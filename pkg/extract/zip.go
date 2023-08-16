package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
)

// base reference: https://golang.cafe/blog/golang-unzip-file-example.html

type Zip struct {
	fileSuffix string
}

func NewZip() *Zip {
	return &Zip{fileSuffix: ".zip"}
}

func (z *Zip) FileSuffix() string {
	return z.fileSuffix
}

func (z *Zip) Extract(e *Extract, src string, dst string) error {

	// open archive
	archive, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer archive.Close()

	var fileCounter int64

	// walk over archive
	for _, archiveFile := range archive.File {

		// check for to many files in archive
		if err := e.incrementAndCheckMaxFiles(&fileCounter); err != nil {
			return err
		}

		hdr := archiveFile.FileHeader

		switch hdr.Mode() & os.ModeType {
		case os.ModeDir:
			// handle directory
			if err := e.createDir(dst, archiveFile.Name); err != nil {
				return err
			}
			continue

		case os.ModeSymlink:
			// handle symlink
			linkTarget, err := readLinkTargetFromZip(archiveFile)
			if err != nil {
				return err
			}
			if err := e.createSymlink(dst, archiveFile.Name, linkTarget); err != nil {
				return err
			}
			continue

		// in case of a normal file the value is not set
		case 0:

			// check for file size
			if err := e.checkFileSize(int64(archiveFile.UncompressedSize64)); err != nil {
				return err
			}

			fileInArchive, err := archiveFile.Open()
			if err != nil {
				return err
			}
			defer func() {
				fileInArchive.Close()
			}()
			// create the file
			if err := e.createFile(dst, archiveFile.Name, fileInArchive, archiveFile.Mode()); err != nil {
				return err
			}

		// catch all for unspported file modes
		default:
			return fmt.Errorf("unspported filemode: %s", hdr.Mode()&os.ModeType)

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
