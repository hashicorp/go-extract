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

// magicBytesZIP contains the magic bytes for a zip archive.
// reference: https://golang.org/pkg/archive/zip/
var magicBytesZIP = [][]byte{
	{0x50, 0x4B, 0x03, 0x04},
}

// IsZip checks if data is a zip archive. It returns true if data is a zip archive and false if data is not a zip archive.
func IsZip(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytesZIP)
}

// Unpack sets a timeout for the ctx and starts the zip extraction from src to dst. It returns an error if the extraction failed.
func UnpackZip(ctx context.Context, src io.Reader, dst string, cfg *config.Config) error {

	// prepare metrics collection and emit
	m := &config.Metrics{ExtractedType: "zip"}
	defer cfg.MetricsHook(ctx, m)
	captureExtractionDuration(ctx, cfg)

	// check if src is a readerAt and an io.Seeker
	if _, ok := src.(io.Seeker); ok {
		if _, ok := src.(io.ReaderAt); ok {
			return unpackZipReaderAtSeeker(ctx, src, dst, cfg, m)
		}
	}

	return unpackZipCached(ctx, src, dst, cfg, m)
}

// unpackZipReaderAtSeeker checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
// src is a readerAt and a seeker. If the InputSize exceeds the maximum input size, the function returns an error.
func unpackZipReaderAtSeeker(ctx context.Context, src io.Reader, dst string, cfg *config.Config, m *config.Metrics) error {

	// log extraction
	cfg.Logger().Info("extracting zip")

	// check if src is a seeker and readerAt
	var s io.Seeker
	var ra io.ReaderAt
	var ok bool
	if s, ok = src.(io.Seeker); !ok {
		return handleError(cfg, m, "cannot convert src to seeker", fmt.Errorf("reader is not a seeker"))
	}
	if ra, ok = src.(io.ReaderAt); !ok {
		return handleError(cfg, m, "cannot convert src to readerAt", fmt.Errorf("reader is not a readerAt"))
	}

	// get size of input and check if it exceeds maximum input size
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return handleError(cfg, m, "cannot seek to end of reader", err)
	}
	m.InputSize = size
	if cfg.MaxInputSize() != -1 && size > cfg.MaxInputSize() {
		return handleError(cfg, m, "cannot unpack zip", fmt.Errorf("input size exceeds maximum input size"))
	}

	// create zip reader and extract
	reader, err := zip.NewReader(ra, size)
	if err != nil {
		return handleError(cfg, m, "cannot create zip reader", err)
	}
	return unpackZip(ctx, reader, dst, cfg, m)
}

// unpackZipCached checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
// It caches the input on disc or in memory before starting extraction. If the input is larger than the maximum input size, the function
// returns an error. If the input is smaller than the maximum input size, the function creates a zip reader and extracts the contents
// to dst.
func unpackZipCached(ctx context.Context, src io.Reader, dst string, cfg *config.Config, m *config.Metrics) error {

	// log caching
	cfg.Logger().Info("caching zip input")

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
		// provide tmpFile as readerAt and seeker
		return unpackZipReaderAtSeeker(ctx, tmpFile, dst, cfg, m)
	}

	// cache src in memory before starting extraction
	data, err := io.ReadAll(ler)
	if err != nil {
		return handleError(cfg, m, "cannot read all from reader", err)
	}
	reader := bytes.NewReader(data)
	return unpackZipReaderAtSeeker(ctx, reader, dst, cfg, m)
}

// unpack checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst. It uses the zip.Reader
// to extract the contents to dst. It checks if the file names match the patterns given in the config. If a file name does not match
// the patterns, the file is skipped. If a file name matches the patterns, the file is extracted to dst. If the file is a directory,
// the directory is created in dst. If the file is a symlink, the symlink is created in dst. If the file is a regular file, the file
// is created in dst. If the file is a unsupported file mode, the file is skipped. If the file is a unsupported file mode and the
// config allows unsupported files, the file is skipped. If the file is a unsupported file mode and the config does not allow unsupported
// files, the function returns an error. If the extraction size exceeds the maximum extraction size, the function returns an error.
// If the extraction size does not exceed the maximum extraction size, the function returns nil.
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
			m.PatternMismatches++
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
			if c.DenySymlinkExtraction() {

				// check for continue for unsupported files
				if c.ContinueOnUnsupportedFiles() {
					m.UnsupportedFiles++
					m.LastUnsupportedFile = hdr.Name
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
				m.UnsupportedFiles++
				m.LastUnsupportedFile = hdr.Name
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

// readLinkTargetFromZip extracts the symlink destination for symlinkFile from the zip archive.
// It returns the symlink destination or an error if the symlink destination could not be extracted.
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
