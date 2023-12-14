package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// base reference: https://golang.cafe/blog/golang-unzip-file-example.html

var MagicBytesZIP = [][]byte{
	{0x50, 0x4B, 0x03, 0x04},
}

// Zip is implements the Extractor interface to extract zip archives.
type Zip struct{}

// NewZip returns a new zip object with config as configuration.
func NewZip() *Zip {
	// instantiate
	zip := Zip{}

	// return
	return &zip
}

// Unpack sets a timeout for the ctx and starts the zip extraction from src to dst.
func (z *Zip) Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
	return z.unpack(ctx, src, dst, t, c)
}

// unpack checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
func (z *Zip) unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
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
	if err := c.CheckMaxFiles(int64(len(zipReader.File))); err != nil {
		return err
	}

	// summarize file-sizes
	var extractionSize uint64

	// walk over archive
	for _, archiveFile := range zipReader.File {

		// check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// get next file
		hdr := archiveFile.FileHeader

		c.Log.Debug("extract", "name", hdr.Name)
		switch hdr.Mode() & os.ModeType {

		case os.ModeDir: // handle directory

			// create dir
			if err := t.CreateSafeDir(c, dst, hdr.Name); err != nil {

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe directory", "error", err)
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
			if err := t.CreateSafeSymlink(c, dst, hdr.Name, linkTarget); err != nil {

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe symlink", "error", err)
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
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				return err
			}

			// open stream
			fileInArchive, err := archiveFile.Open()
			if err != nil {
				return err
			}

			// create the file
			if err := t.CreateSafeFile(c, dst, hdr.Name, fileInArchive, archiveFile.Mode()); err != nil {

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe directory", "error", err)
					fileInArchive.Close()
					continue
				}

				// end extraction
				fileInArchive.Close()
				return err
			}

			// next item
			fileInArchive.Close()
			continue
		default: // catch all for unsupported file modes
			return fmt.Errorf("unsupported filetype in archive")
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
