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
			c.MetricsHook(ctx, metrics)
		}
	}()

	// read src into buffer and check for limits and errors
	buff := bytes.NewBuffer([]byte{})
	var size int64
	{
		var err error
		if c.MaxExtractionSize > -1 {
			// read src safe into buffer
			for {
				buf := make([]byte, 1024)
				n, err := src.Read(buf)
				if err != nil && err != io.EOF {
					msg := "cannot read source zip archive"
					return handleError(c, &metrics, msg, err)
				}

				// clothing read
				if n == 0 {
					break
				}

				// check if maximum is exceeded
				if size+int64(n) < c.MaxExtractionSize {
					buff.Write(buf[:n])
					size = size + int64(n)

					// check if context is errored
					if ctx.Err() != nil {
						msg := "context error"
						return handleError(c, &metrics, msg, ctx.Err())
					}
				} else {
					err := fmt.Errorf("archive size exceeds maximum extraction size")
					msg := "cannot continue decompress zip"
					return handleError(c, &metrics, msg, err)
				}
			}
		} else {
			// read all without limitations
			size, err = io.Copy(buff, src)
		}
		if err != nil && err != io.EOF {
			msg := "cannot read src"
			return handleError(c, &metrics, msg, err)
		}
	}

	// get content of buf as io.Reader
	reader := bytes.NewReader(buff.Bytes())

	// Open a zip archive for reading.
	zipReader, err := zip.NewReader(reader, size)

	// check for errors, format and handle them
	if err != nil {
		msg := "cannot read zip"
		return handleError(c, &metrics, msg, err)
	}

	// check for to many files in archive
	if err := c.CheckMaxObjects(int64(len(zipReader.File))); err != nil {
		msg := "max objects check failed"
		return handleError(c, &metrics, msg, err)
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
			return handleError(c, &metrics, "max objects check failed", err)
		}

		c.Logger.Info("extract", "name", hdr.Name)
		switch hdr.Mode() & os.ModeType {

		case os.ModeDir: // handle directory

			// check if dir is just current working dir
			if filepath.Clean(hdr.Name) == "." {
				continue
			}

			// create dir and check for errors, format and handle them
			if err := t.CreateSafeDir(c, dst, hdr.Name); err != nil {
				msg := "failed to create safe directory"
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// next item
			metrics.ExtractedDirs++
			continue

		case os.ModeSymlink: // handle symlink

			// extract link target
			linkTarget, err := readLinkTargetFromZip(archiveFile)

			// check for errors, format and handle them
			if err != nil {
				msg := "failed to read symlink target"
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// step over creation
				continue
			}

			// create link
			if err := t.CreateSafeSymlink(c, dst, hdr.Name, linkTarget); err != nil {
				msg := "failed to create safe symlink"
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// next item
			metrics.ExtractedSymlinks++
			continue

		case 0: // handle regular files

			// check for file size
			extractionSize = extractionSize + archiveFile.UncompressedSize64
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				msg := "maximum extraction size exceeded"
				return handleError(c, &metrics, msg, err)
			}

			// open stream
			fileInArchive, err := archiveFile.Open()

			// check for errors, format and handle them
			if err != nil {
				msg := "cannot open file in archive"
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// create the file
			if err := t.CreateSafeFile(c, dst, hdr.Name, fileInArchive, archiveFile.Mode()); err != nil {
				msg := "failed to create safe file"
				if err := handleError(c, &metrics, msg, err); err != nil {
					fileInArchive.Close()
					return err
				}

				// don't collect metrics on failure
				fileInArchive.Close()
				continue
			}

			// next item
			metrics.ExtractionSize = int64(extractionSize)
			metrics.ExtractedFiles++
			fileInArchive.Close()
			continue

		default: // catch all for unsupported file modes

			// increase error counter, set error and end if necessary
			err := fmt.Errorf("unsupported file mode (%x)", hdr.Mode())
			msg := "cannot extract file"
			if err := handleError(c, &metrics, msg, err); err != nil {
				return err
			}

			continue
		}
	}

	// finished without problems
	return nil
}

// handleError increases the error counter, sets the latest error and
// decides if extraction should continue.
func handleError(c *config.Config, metrics *config.Metrics, msg string, err error) error {

	// increase error counter and set error
	metrics.ExtractionErrors++
	metrics.LastExtractionError = err

	// do not end on error
	if c.ContinueOnError {
		c.Logger.Error(msg, "error", err)
		return nil
	}

	// end extraction on error
	return fmt.Errorf("%s: %s", msg, err)
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
