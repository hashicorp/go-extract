package extractor

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// base reference: https://golang.cafe/blog/golang-unzip-file-example.html

type Zip struct {
	config     *config.Config
	fileSuffix string
}

func NewZip(config *config.Config) *Zip {
	return &Zip{fileSuffix: ".zip", config: config}
}

func (z *Zip) FileSuffix() string {
	return z.fileSuffix
}

func (z *Zip) Unpack(ctx context.Context, src string, dst string) error {

	target := &target.Os{}

	// open zipFile
	zipFile, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// check for to many files in archive
	if err := z.config.CheckMaxFiles(int64(len(zipFile.File))); err != nil {
		return err
	}

	// walk over archive
	for _, archiveFile := range zipFile.File {

		hdr := archiveFile.FileHeader

		switch hdr.Mode() & os.ModeType {
		case os.ModeDir:
			// handle directory
			if err := target.CreateSafeDir(dst, archiveFile.Name); err != nil {
				return err
			}
			continue

		case os.ModeSymlink:
			// handle symlink
			linkTarget, err := readLinkTargetFromZip(archiveFile)
			if err != nil {
				return err
			}
			if err := target.CreateSafeSymlink(dst, archiveFile.Name, linkTarget); err != nil {
				return err
			}
			continue

		// in case of a normal file the value is not set
		case 0:

			// check for file size
			if err := z.config.CheckFileSize(int64(archiveFile.UncompressedSize64)); err != nil {
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
			if err := target.CreateSafeFile(z.config, dst, archiveFile.Name, fileInArchive, archiveFile.Mode()); err != nil {
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
