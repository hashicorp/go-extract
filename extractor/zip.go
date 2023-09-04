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

// Zip is implements the Extractor interface to extract zip archives.
type Zip struct {
	config     *config.Config
	fileSuffix string
	target     target.Target
	magicBytes [][]byte
	offset     int
}

// NewZip returns a new zip object with config as configuration.
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
		fileSuffix: fileSuffix,
		config:     config,
		target:     target,
		magicBytes: magicBytes,
		offset:     offset,
	}

	// return
	return &zip
}

// FileSuffix returns the common file suffix of zip archive type.
func (z *Zip) FileSuffix() string {
	return z.fileSuffix
}

// SetConfig sets config as configuration.
func (z *Zip) SetConfig(config *config.Config) {
	z.config = config
}

// Offset returns the offset for the magic bytes.
func (z *Zip) Offset() int {
	return z.offset
}

// MagicBytes returns the magic bytes that identifies zip files.
func (z *Zip) MagicBytes() [][]byte {
	return z.magicBytes
}

// Unpack sets a timeout for the ctx and starts the zip extraction from src to dst.
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

// SetTarget sets target as a extraction destination
func (z *Zip) SetTarget(target target.Target) {
	z.target = target
}

// unpack checks ctx for cancelation, while it reads a zip file from src and extracts the contents to dst.
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

		z.config.Log.Printf("extract %s", hdr.Name)
		switch hdr.Mode() & os.ModeType {

		case os.ModeDir: // handle directory

			// create dir
			if err := z.target.CreateSafeDir(z.config, dst, hdr.Name); err != nil {

				// do not end on error
				if z.config.ContinueOnError {
					z.config.Log.Printf("extraction error: %s", err)
					continue
				}

				// end extraction
				return err
			}

			// next item
			continue

		case os.ModeSymlink: // handle symlink

			// extract link target
			linkTarget, err := readLinkTargetFromZip(archiveFile)
			if err != nil {
				return err
			}

			// create link
			if err := z.target.CreateSafeSymlink(z.config, dst, hdr.Name, linkTarget); err != nil {

				// do not end on error
				if z.config.ContinueOnError {
					z.config.Log.Printf("extraction error: %s", err)
					continue
				}

				// end extraction
				return err
			}

			// next item
			continue

		case 0: // handle regular files

			// check for file size
			extractionSize = extractionSize + archiveFile.UncompressedSize64
			if err := z.config.CheckExtractionSize(int64(extractionSize)); err != nil {
				return err
			}

			// open stream
			fileInArchive, err := archiveFile.Open()
			if err != nil {
				return err
			}
			defer func() {
				fileInArchive.Close()
			}()

			// create the file
			if err := z.target.CreateSafeFile(z.config, dst, hdr.Name, fileInArchive, archiveFile.Mode()); err != nil {

				// do not end on error
				if z.config.ContinueOnError {
					z.config.Log.Printf("extraction error: %s", err)
					continue
				}

				// end extraction
				return err
			}

			// next item
			continue

		default: // catch all for unspported file modes

			// drop error
			return fmt.Errorf("unspported filetype in archive.")

		}
	}

	// finished without problems
	return nil
}

// readLinkTargetFromZip extracts the symlink destination for symlinkFile
func readLinkTargetFromZip(symlinkFile *zip.File) (string, error) {

	// read content to determine symlink destination
	rc, err := symlinkFile.Open()
	if err != nil {
		return "", err
	}
	defer func() {
		rc.Close()
	}()

	// read link target
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	symlinkTarget := string(data)

	// return result
	return symlinkTarget, nil
}
