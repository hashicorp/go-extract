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

	// prepare metrics collection and emit
	m := &config.Metrics{ExtractedType: "zip"}
	defer cfg.MetricsHook(ctx, m)
	captureExtractionDuration(ctx, cfg)

	// check if src is a file, if so - use it directly
	if inf, ok := src.(*os.File); ok {
		return z.unpackZipFile(ctx, inf, dst, t, cfg, m)
	}

	return z.unpackZipReader(ctx, src, dst, t, cfg, m)
}

func (z *Zip) unpackZipReader(ctx context.Context, src io.Reader, dst string, t target.Target, cfg *config.Config, m *config.Metrics) error {

	// check if src is a readerAt and an io.Seeker
	if seeker, ok := src.(io.Seeker); ok {
		if ra, ok := src.(io.ReaderAt); ok {
			size, err := seeker.Seek(0, io.SeekEnd)
			if err != nil {
				return handleError(cfg, m, "cannot seek to end of reader", err)
			}
			// create zip reader
			reader, err := zip.NewReader(ra, size)
			if err != nil {
				return handleError(cfg, m, "cannot create zip reader", err)
			}

			// perform extraction
			return z.unpack(ctx, reader, dst, t, cfg, m)
		}
	}

	// create limit error reader for src
	ler := config.NewLimitErrorReader(src, cfg.MaxInputSize())

	// cache src in temp file for extraction
	if !cfg.CacheInMemory() {
		// copy src to tmp file
		tmpFile, err := os.CreateTemp("", "extractor-*.zip")
		if err != nil {
			return handleError(cfg, m, "cannot create tmp file", err)
		}
		defer tmpFile.Close()
		defer os.Remove(tmpFile.Name())
		if _, err := io.Copy(tmpFile, ler); err != nil {
			return handleError(cfg, m, "cannot copy reader to file", err)
		}
		return z.unpackZipFile(ctx, tmpFile, dst, t, cfg, m)
	}

	// cache src in memory before starting extraction
	data, err := io.ReadAll(ler)
	if err != nil {
		return handleError(cfg, m, "cannot read all from reader", err)
	}
	m.InputSize = int64(len(data))
	// create zip reader
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return handleError(cfg, m, "cannot create zip reader", err)
	}
	// perform extraction
	return z.unpack(ctx, reader, dst, t, cfg, m)
}

// unpackZipFile checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
// It also sets the input size in the metrics.
func (z *Zip) unpackZipFile(ctx context.Context, src *os.File, dst string, t target.Target, cfg *config.Config, m *config.Metrics) error {
	// get file size
	stat, err := src.Stat()
	if err != nil {
		return handleError(cfg, m, "cannot stat file", err)
	}
	m.InputSize = stat.Size()

	// check for maximum input size
	if cfg.MaxInputSize() != -1 && stat.Size() > cfg.MaxInputSize() {
		return handleError(cfg, m, "cannot unpack zip", fmt.Errorf("input size exceeds maximum input size"))
	}

	// open zip file
	reader, err := zip.OpenReader(src.Name())
	if err != nil {
		return handleError(cfg, m, "cannot open zip file", err)
	}
	defer reader.Close()

	// perform extraction
	return z.unpack(ctx, &reader.Reader, dst, t, cfg, m)
}

// unpack checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
func (z *Zip) unpack(ctx context.Context, src *zip.Reader, dst string, t target.Target, c *config.Config, m *config.Metrics) error {

	// check for to many files in archive
	if err := c.CheckMaxObjects(int64(len(src.File))); err != nil {
		return handleError(c, m, "max objects check failed", err)
	}

	// summarize file-sizes
	var extractionSize uint64

	// walk over archive
	for _, archiveFile := range src.File {

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
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// next item
			m.ExtractionSize = int64(extractionSize)
			m.ExtractedFiles++
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
