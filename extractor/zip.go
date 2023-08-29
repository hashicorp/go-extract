package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// base reference: https://golang.cafe/blog/golang-unzip-file-example.html

type Zip struct {
	config     *config.Config
	fileSuffix string
	target     target.Target
	magicBytes [][]byte
	offset     int
}

func NewZip(config *config.Config) *Zip {
	// defaults
	const (
		fileSuffix = ".zip"
	)
	target := target.NewOs()
	magicBytes := [][]byte{
		{0x50, 0x4B, 0x03, 0x04},
	}
	offset := 0

	// instantiate
	zip := Zip{
		fileSuffix: ".zip",
		config:     config,
		target:     &target,
		magicBytes: magicBytes,
		offset:     offset,
	}

	// return
	return &zip
}

func (z *Zip) FileSuffix() string {
	return z.fileSuffix
}

func (z *Zip) SetConfig(config *config.Config) {
	z.config = config
}

func (z *Zip) Offset() int {
	return z.offset
}

func (z *Zip) MagicBytes() [][]byte {
	return z.magicBytes
}

func (z *Zip) Unpack(ctx context.Context, src io.Reader, dst string) error {

	// start extraction without timer
	if z.config.MaxExtractionTime == -1 {
		return z.unpack(ctx, src, dst)
	}

	// prepare timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(z.config.MaxExtractionTime)*time.Second)
	defer cancel()

	exChan := make(chan error, 1)
	go func() {
		if err := z.unpack(ctx, src, dst); err != nil {
			exChan <- err
		}
		exChan <- nil
	}()

	// start extraction in on thread
	select {
	case err := <-exChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("maximum extraction time exceeded")
	}

	return nil
}

func (z *Zip) SetTarget(target *target.Target) {
	z.target = *target
}

func (z *Zip) unpack(ctx context.Context, src io.Reader, dst string) error {

	// convert io.Reader to io.ReaderAt
	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, src)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buff.Bytes())

	// Open a zip archive for reading.
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return err
	}

	// check for to many files in archive
	if err := z.config.CheckMaxFiles(int64(len(zipReader.File))); err != nil {
		return err
	}

	// summarize filesizes
	var extractionSize uint64

	// walk over archive
	for _, archiveFile := range zipReader.File {

		// check if context is cancled
		if ctx.Err() != nil {
			return nil
		}

		// get next file
		hdr := archiveFile.FileHeader

		switch hdr.Mode() & os.ModeType {
		case os.ModeDir:
			// handle directory
			if err := z.target.CreateSafeDir(z.config, dst, archiveFile.Name); err != nil {
				return err
			}
			continue

		case os.ModeSymlink:
			// handle symlink
			linkTarget, err := readLinkTargetFromZip(archiveFile)
			if err != nil {
				return err
			}
			if err := z.target.CreateSafeSymlink(z.config, dst, archiveFile.Name, linkTarget); err != nil {
				return err
			}
			continue

		// in case of a normal file the value is not set
		case 0:

			// check for file size
			extractionSize = extractionSize + archiveFile.UncompressedSize64
			if err := z.config.CheckExtractionSize(int64(extractionSize)); err != nil {
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
			if err := z.target.CreateSafeFile(z.config, dst, archiveFile.Name, fileInArchive, archiveFile.Mode()); err != nil {
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
