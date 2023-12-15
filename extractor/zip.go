package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

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

	// object to store metrics
	metrics := config.Metrics{}
	metrics.ExtractedType = "zip"
	start := time.Now()

	// anonymous function to emit metrics
	defer func() {

		// calculate execution time
		metrics.ExtractionDuration = time.Since(start)

		// emit metrics
		if c.MetricsHook != nil {
			c.MetricsHook(metrics)
		}
	}()

	// convert io.Reader to io.ReaderAt
	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, src)
	if err != nil {
		metrics.ExtractionErrors++
		metrics.LastExtractionError = fmt.Errorf("cannot read src: %s", err)
		return metrics.LastExtractionError
	}
	reader := bytes.NewReader(buff.Bytes())

	// Open a zip archive for reading.
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		metrics.ExtractionErrors++
		metrics.LastExtractionError = fmt.Errorf("cannot read zip: %s", err)
		return metrics.LastExtractionError
	}

	// check for to many files in archive
	if err := c.CheckMaxObjects(int64(len(zipReader.File))); err != nil {
		metrics.ExtractionErrors++
		metrics.LastExtractionError = err
		return err
	}

	// summarize file-sizes
	var extractionSize uint64
	var objectCounter int64

	// walk over archive
	for _, archiveFile := range zipReader.File {

		// check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// get next file
		hdr := archiveFile.FileHeader

		// check for to many objects in archive
		objectCounter++

		// check if maximum of objects is exceeded
		if err := c.CheckMaxObjects(objectCounter); err != nil {
			metrics.ExtractionErrors++
			metrics.LastExtractionError = err
			return err
		}

		c.Log.Debug("extract", "name", hdr.Name)
		switch hdr.Mode() & os.ModeType {

		case os.ModeDir: // handle directory

			// check if dir is just current working dir
			if filepath.Clean(hdr.Name) == "." {
				continue
			}

			// create dir
			if err := t.CreateSafeDir(c, dst, hdr.Name); err != nil {

				metrics.ExtractionErrors++
				metrics.LastExtractionError = err

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe directory", "error", err)
					continue
				}

				// end extraction
				return err
			}

			// next item
			metrics.ExtractedDirs++
			continue

		case os.ModeSymlink: // handle symlink

			// extract link target
			linkTarget, err := readLinkTargetFromZip(archiveFile)
			if err != nil {
				metrics.ExtractionErrors++
				metrics.LastExtractionError = err

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to read symlink target", "error", err)
					continue
				}

				return err
			}

			// create link
			if err := t.CreateSafeSymlink(c, dst, hdr.Name, linkTarget); err != nil {

				metrics.ExtractionErrors++
				metrics.LastExtractionError = err

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe symlink", "error", err)
					continue
				}

				// end extraction
				return err
			}

			// next item
			metrics.ExtractedSymlinks++
			continue

		case 0: // handle regular files

			// check for file size
			extractionSize = extractionSize + archiveFile.UncompressedSize64
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				metrics.ExtractionErrors++
				metrics.LastExtractionError = fmt.Errorf("maximum extraction size exceeded: %s", err)
				return metrics.LastExtractionError
			}

			// open stream
			fileInArchive, err := archiveFile.Open()
			if err != nil {
				metrics.ExtractionErrors++
				metrics.LastExtractionError = fmt.Errorf("cannot open file in archive: %s", err)
				return metrics.LastExtractionError
			}

			// create the file
			if err := t.CreateSafeFile(c, dst, hdr.Name, fileInArchive, archiveFile.Mode()); err != nil {

				metrics.ExtractionErrors++
				metrics.LastExtractionError = err

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe file", "error", err)
					fileInArchive.Close()
					continue
				}

				// end extraction
				fileInArchive.Close()
				return err
			}

			// next item
			metrics.ExtractionSize = int64(extractionSize)
			metrics.ExtractedFiles++
			fileInArchive.Close()
			continue
		default: // catch all for unsupported file modes
			metrics.ExtractionErrors++
			metrics.LastExtractionError = fmt.Errorf("unsupported file mode: %s", hdr.Mode())
			return metrics.LastExtractionError
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
