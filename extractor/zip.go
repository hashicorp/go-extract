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

	// ensure input size and capture metrics
	ler := NewLimitErrorReader(src, c.MaxInputSize)
	src = ler

	// object to store metrics
	metrics := config.Metrics{}
	metrics.ExtractedType = "zip"
	start := time.Now()

	// anonymous function to emit metrics
	defer func() {

		// store input file size
		metrics.InputSize = ler.N

		// calculate execution time
		metrics.ExtractionDuration = time.Since(start)

		// emit metrics
		if c.MetricsHook != nil {
			c.MetricsHook(ctx, metrics)
		}
	}()

	// read complete zip file into memory
	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, src)
	if err != nil {
		msg := "cannot read src"
		return handleError(c, &metrics, msg, err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		msg := "context error"
		return handleError(c, &metrics, msg, err)
	}

	// get content of buf as io.Reader
	srcReader := bytes.NewReader(buff.Bytes())
	zipReader, err := zip.NewReader(srcReader, size)

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

// limitErrorReader is a reader that returns an error if the limit is exceeded
// before the underlying reader is fully read.
// If the limit is -1, all data from the original reader is read.
type limitErrorReader struct {
	R io.Reader // underlying reader
	L int64     // limit
	N int64     // number of bytes read
}

// Read reads from the underlying reader and fills up p.
// It returns an error if the limit is exceeded, even if the underlying reader is not fully read.
// If the limit is -1, all data from the original reader is read.
// Remark: Even if the limit is exceeded, the buffer p is filled up to the max or until the underlying
// reader is fully read.
func (l *limitErrorReader) Read(p []byte) (int, error) {

	if l.L == -1 {
		return l.R.Read(p)
	}

	// read from underlying reader
	n, err := l.R.Read(p)
	l.N += int64(n)
	if err != nil {
		return n, err
	}

	// check if limit has exceeded
	if l.N > l.L {
		return n, fmt.Errorf("read limit exceeded")
	}

	// return
	return n, err
}

// NewLimitErrorReader returns a new LimitErrorReader that reads from r
func NewLimitErrorReader(r io.Reader, limit int64) *limitErrorReader {
	return &limitErrorReader{R: r, L: limit, N: 0}
}
