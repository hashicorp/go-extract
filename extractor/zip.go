package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// base reference: https://golang.cafe/blog/golang-unzip-file-example.html

var magicBytesZIP = [][]byte{
	{0x50, 0x4B, 0x03, 0x04},
}

// Zip is implements the Extractor interface to extract zip archives.
type Zip struct{}

func IsZip(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytesZIP)
}

// NewZip returns a new zip object with config as configuration.
func NewZip() *Zip {
	// instantiate
	zip := Zip{}

	// return
	return &zip
}

// Unpack sets a timeout for the ctx and starts the zip extraction from src to dst.
func (z *Zip) Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {

	// ensures extraction time is capturing
	captureExtractionDuration(ctx, c)

	// prepare extraction
	reader, inputSize, err := readerToReaderAt(src, c.MaxInputSize())
	if err != nil {
		return handleError(c, nil, "cannot read all from reader", err)
	}

	// check for maximum input size
	if c.MaxInputSize() != -1 && inputSize > c.MaxInputSize() {
		return fmt.Errorf("file size exceeds maximum input size")
	} else {
		// setup metric hook
		c.AddMetricsProcessor(func(ctx context.Context, m *config.Metrics) {
			m.InputSize = inputSize
		})
	}

	// perform extraction
	return z.unpack(ctx, reader, dst, t, c, inputSize)
}

// unpack checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
func (z *Zip) unpack(ctx context.Context, src io.ReaderAt, dst string, t target.Target, c *config.Config, inputSize int64) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: "zip", InputSize: inputSize}

	// emit metrics
	defer c.MetricsHook(ctx, &metrics)

	// get content of readerAt as io.Reader
	zipReader, err := zip.NewReader(src, inputSize)

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
			msg := "max objects exceeded"
			if err := handleError(c, &metrics, msg, err); err != nil {
				return err
			}

			// go to next item
			continue
		}

		c.Logger().Info("extract", "name", hdr.Name)
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

			// check if unsupported files should be skipped
			if c.ContinueOnUnsupportedFiles() {
				metrics.SkippedUnsupportedFiles++
				metrics.LastSkippedUnsupportedFile = hdr.Name
				continue
			}

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

func readIntoTmpFile(src io.Reader) (string, error) {
	// create tmp file
	tmpFile, err := os.CreateTemp("", "extractor-*.zip")
	if err != nil {
		return "", fmt.Errorf("cannot create tmp file: %w", err)
	}

	// copy src to tmp file
	if _, err := io.Copy(tmpFile, src); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("cannot copy src to tmp file: %w", err)
	}

	// close tmp file
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("cannot close tmp file: %w", err)
	}

	// return tmp file
	return tmpFile.Name(), nil
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

// readerToReaderAt converts a reader to a readerAt
func readerToReaderAt(src io.Reader, inputLimit int64) (io.ReaderAt, int64, error) {

	// 	check if src is a seekable reader and offers io.ReadAt
	if s, ok := src.(io.Seeker); ok {
		if r, ok := src.(io.ReaderAt); ok {
			// get file size
			size, err := s.Seek(0, io.SeekEnd)
			if err != nil {
				return nil, 0, fmt.Errorf("cannot seek to end of reader: %w", err)
			}
			return r, size, nil
		}
	}

	// read file into memory
	ler := config.NewLimitErrorReader(src, inputLimit)
	data, err := io.ReadAll(ler)
	if err != nil {
		return nil, int64(len(data)), fmt.Errorf("cannot copy reader to buffer: %w", err)
	}
	return bytes.NewReader(data), int64(len(data)), nil
}
