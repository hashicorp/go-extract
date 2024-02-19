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
)

// base reference: https://golang.cafe/blog/golang-unzip-file-example.html

var magicBytesZIP = [][]byte{
	{0x50, 0x4B, 0x03, 0x04},
}

// IsZip checks if data is a zip archive.
func IsZip(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytesZIP)
}

// Unpack sets a timeout for the ctx and starts the zip extraction from src to dst.
func UnpackZip(ctx context.Context, src io.Reader, dst string, cfg *config.Config) error {

	// prepare metrics collection and emit
	m := &config.Metrics{ExtractedType: "zip"}
	defer cfg.MetricsHook(ctx, m)
	captureExtractionDuration(ctx, cfg)

	// check if src is a file, if so - use it directly
	if inf, ok := src.(*os.File); ok {
		return unpackZipFile(ctx, inf, dst, cfg, m)
	}

	return unpackZipReader(ctx, src, dst, cfg, m)
}

func unpackZipReader(ctx context.Context, src io.Reader, dst string, cfg *config.Config, m *config.Metrics) error {

	// check if src is a readerAt and an io.Seeker
	if seeker, ok := src.(io.Seeker); ok {
		if ra, ok := src.(io.ReaderAt); ok {
			size, err := seeker.Seek(0, io.SeekEnd)
			if err != nil {
				return handleError(cfg, m, "cannot seek to end of reader", err)
			}
			m.InputSize = size
			if cfg.MaxInputSize() != -1 && size > cfg.MaxInputSize() {
				return handleError(cfg, m, "cannot unpack zip", fmt.Errorf("input size exceeds maximum input size"))
			}
			// create zip reader
			reader, err := zip.NewReader(ra, size)
			if err != nil {
				return handleError(cfg, m, "cannot create zip reader", err)
			}

			// perform extraction
			return unpackZip(ctx, reader, dst, cfg, m)
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
		return unpackZipFile(ctx, tmpFile, dst, cfg, m)
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
	return unpackZip(ctx, reader, dst, cfg, m)
}

// unpackZipFile checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
// It also sets the input size in the metrics.
func unpackZipFile(ctx context.Context, src *os.File, dst string, cfg *config.Config, m *config.Metrics) error {
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
	return unpackZip(ctx, &reader.Reader, dst, cfg, m)
}

// unpack checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
func unpackZip(ctx context.Context, src *zip.Reader, dst string, c *config.Config, m *config.Metrics) error {

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
			return handleError(c, m, "cannot check pattern", err)
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
			if err := unpackTarget.CreateSafeDir(c, dst, hdr.Name); err != nil {
				if err := handleError(c, m, "failed to create safe directory", err); err != nil {
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

				if err := handleError(c, m, "cannot extract symlink", fmt.Errorf("symlinks are not allowed")); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// extract link target
			linkTarget, err := readLinkTargetFromZip(archiveFile)

			// check for errors, format and handle them
			if err != nil {
				if err := handleError(c, m, "failed to read symlink target", err); err != nil {
					return err
				}

				// step over creation
				continue
			}

			// create link
			if err := unpackTarget.CreateSafeSymlink(c, dst, hdr.Name, linkTarget); err != nil {
				if err := handleError(c, m, "failed to create safe symlink", err); err != nil {
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
				return handleError(c, m, "maximum extraction size exceeded", err)
			}

			// open stream
			fileInArchive, err := archiveFile.Open()
			defer fileInArchive.Close()

			// check for errors, format and handle them
			if err != nil {
				if err := handleError(c, m, "cannot open file in archive", err); err != nil {
					return err
				}

				// don't collect metrics on failure
				continue
			}

			// create the file
			if err := unpackTarget.CreateSafeFile(c, dst, hdr.Name, fileInArchive, archiveFile.Mode()); err != nil {
				if err := handleError(c, m, "failed to create safe file", err); err != nil {
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
			if err := handleError(c, m, "cannot extract file", fmt.Errorf("unsupported file mode (%x)", hdr.Mode())); err != nil {
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
