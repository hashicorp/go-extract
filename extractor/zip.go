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

// IsZip checks if data is a zip archive.
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
func (z *Zip) Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, cfg *config.Config) error {

	// object to store m
	m := &config.Metrics{ExtractedType: "zip"}

	// emit metrics
	defer cfg.MetricsHook(ctx, m)

	// ensures extraction time is capturing
	captureExtractionDuration(ctx, cfg)

	// prepare extraction
	reader, inputSize, tmpFile, err := readerToReaderAt(src, cfg)
	// clean up tmpFile
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}
	}()
	if err != nil {
		return handleError(cfg, m, "cannot read all from reader", err)
	}

	// check for maximum input size
	if cfg.MaxInputSize() != -1 && inputSize > cfg.MaxInputSize() {
		return handleError(cfg, m, "file size exceeds maximum input size", err)
	}

	// setup metric hook
	cfg.AddMetricsProcessor(func(ctx context.Context, m *config.Metrics) {
		m.InputSize = inputSize
	})

	// perform extraction
	return z.unpack(ctx, reader, dst, t, cfg, m, inputSize)
}

// unpack checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
func (z *Zip) unpack(ctx context.Context, src io.ReaderAt, dst string, t target.Target, c *config.Config, m *config.Metrics, inputSize int64) error {

	// get content of readerAt as io.Reader
	zipReader, err := zip.NewReader(src, inputSize)

	// check for errors, format and handle them
	if err != nil {
		msg := "cannot read zip"
		return handleError(c, m, msg, err)
	}

	// check for to many files in archive
	if err := c.CheckMaxObjects(int64(len(zipReader.File))); err != nil {
		msg := "max objects check failed"
		return handleError(c, m, msg, err)
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

		// check if file needs to match patterns
		match, err := checkPatterns(c.Patterns(), hdr.Name)
		if err != nil {
			msg := "cannot check pattern"
			return handleError(c, m, msg, err)
		}
		if !match {
			c.Logger().Info("skipping file (pattern mismatch)", "name", hdr.Name)
			m.SkippedFiles++
			m.LastSkippedFile = hdr.Name
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
				if err := handleError(c, m, msg, err); err != nil {
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// next item
			m.ExtractedDirs++
			continue

		case os.ModeSymlink: // handle symlink

			// check if symlinks are allowed
			if !c.AllowSymlinks() {

				// check for continue for unsupported files
				if c.ContinueOnUnsupportedFiles() {
					m.SkippedUnsupportedFiles++
					m.LastSkippedUnsupportedFile = hdr.Name
					continue
				}

				msg := "symlinks are not allowed"
				err := fmt.Errorf("symlinks are not allowed")
				if err := handleError(c, m, msg, err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// extract link target
			linkTarget, err := readLinkTargetFromZip(archiveFile)

			// check for errors, format and handle them
			if err != nil {
				msg := "failed to read symlink target"
				if err := handleError(c, m, msg, err); err != nil {
					return err
				}

				// step over creation
				continue
			}

			// create link
			if err := t.CreateSafeSymlink(c, dst, hdr.Name, linkTarget); err != nil {
				msg := "failed to create safe symlink"
				if err := handleError(c, m, msg, err); err != nil {
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// next item
			m.ExtractedSymlinks++
			continue

		case 0: // handle regular files

			// check for file size
			extractionSize = extractionSize + archiveFile.UncompressedSize64
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				msg := "maximum extraction size exceeded"
				return handleError(c, m, msg, err)
			}

			// open stream
			fileInArchive, err := archiveFile.Open()
			defer fileInArchive.Close()

			// check for errors, format and handle them
			if err != nil {
				msg := "cannot open file in archive"
				if err := handleError(c, m, msg, err); err != nil {
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// create the file
			if err := t.CreateSafeFile(c, dst, hdr.Name, fileInArchive, archiveFile.Mode()); err != nil {
				msg := "failed to create safe file"
				if err := handleError(c, m, msg, err); err != nil {
					fileInArchive.Close()
					return err
				}

				// don't collect metrics on failure
				fileInArchive.Close()
				continue
			}

			// next item
			m.ExtractionSize = int64(extractionSize)
			m.ExtractedFiles++
			fileInArchive.Close()
			continue

		default: // catch all for unsupported file modes

			// check if unsupported files should be skipped
			if c.ContinueOnUnsupportedFiles() {
				m.SkippedUnsupportedFiles++
				m.LastSkippedUnsupportedFile = hdr.Name
				continue
			}

			// increase error counter, set error and end if necessary
			err := fmt.Errorf("unsupported file mode (%x)", hdr.Mode())
			msg := "cannot extract file"
			if err := handleError(c, m, msg, err); err != nil {
				return err
			}
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
		return "", fmt.Errorf("cannot open file in archive: %s", err)
	}
	defer func() {
		rc.Close()
	}()

	// read link target
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("cannot read symlink target: %s", err)
	}
	symlinkTarget := string(data)

	// return result
	return symlinkTarget, nil
}

// readerToReaderAt converts a reader to a readerAt
func readerToReaderAt(src io.Reader, cfg *config.Config) (io.ReaderAt, int64, *os.File, error) {

	// 	check if src is a seekable reader and offers io.ReadAt
	if s, ok := src.(io.Seeker); ok {
		if r, ok := src.(io.ReaderAt); ok {
			// get file size
			size, err := s.Seek(0, io.SeekEnd)
			if err != nil {
				return nil, 0, nil, fmt.Errorf("cannot seek to end of reader: %s", err)
			}
			return r, size, nil, nil
		}
	}

	// read file into memory
	ler := config.NewLimitErrorReader(src, cfg.MaxInputSize())

	// check if in memory caching is enabled
	if cfg.CacheInMemory() {
		data, err := io.ReadAll(ler)
		if err != nil {
			return nil, int64(len(data)), nil, fmt.Errorf("cannot copy reader to buffer: %s", err)
		}
		return bytes.NewReader(data), int64(len(data)), nil, nil
	}

	// create tmp file
	f, err := os.CreateTemp("", "extractor-*.zip")
	if err != nil {
		return nil, 0, f, fmt.Errorf("cannot create tmp file: %s", err)
	}

	// copy ler into file
	size, err := io.Copy(f, ler)
	if err != nil {
		f.Close()
		return nil, 0, f, fmt.Errorf("cannot copy reader to file: %s", err)
	}

	// reset file
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, 0, f, fmt.Errorf("cannot seek to start of file: %s", err)
	}

	// return adjusted reader
	return f, size, f, nil
}
