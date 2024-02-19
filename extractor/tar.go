package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
)

const offsetTar = 257

var magicBytesTar = [][]byte{
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x20, 0x00},
}

// Tar holds information that are needed for tar extraction.
type Tar struct{}

func IsTar(data []byte) bool {
	return matchesMagicBytes(data, offsetTar, magicBytesTar)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func UnpackTar(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// prepare limits input and ensures metrics capturing
	reader := prepare(ctx, src, c)

	return unpackTar(ctx, reader, dst, c)
}

// unpack checks ctx for cancellation, while it reads a tar file from src and extracts the contents to dst.
func unpackTar(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store m
	m := &config.Metrics{ExtractedType: "tar"}

	// anonymous function to emit metrics
	defer c.MetricsHook(ctx, m)

	// start extraction
	c.Logger().Info("extracting tar")
	var objectCounter int64
	var extractionSize uint64
	tr := tar.NewReader(src)

	// walk through tar
	for {
		// check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// get next file
		hdr, err := tr.Next()

		switch {

		// if no more files are found exit loop
		case err == io.EOF:
			// extraction finished
			return nil

		// return any other error
		case err != nil:
			return handleError(c, m, "error reading tar", err)

		// if the header is nil, just skip it (not sure how this happens)
		case hdr == nil:
			continue
		}

		// check for to many objects in archive
		objectCounter++

		// check if maximum of objects is exceeded
		if err := c.CheckMaxObjects(objectCounter); err != nil {
			return handleError(c, m, "max objects check failed", err)
		}

		// check if name is just current working dir
		if filepath.Clean(hdr.Name) == "." {
			continue
		}

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

		c.Logger().Debug("extract", "name", hdr.Name)
		switch hdr.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:

			// handle directory
			if err := unpackTarget.CreateSafeDir(c, dst, hdr.Name); err != nil {
				if err := handleError(c, m, "failed to create safe directory", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store metrics and continue
			m.ExtractedDirs++
			continue

		// if it's a file create it
		case tar.TypeReg:

			// check extraction size
			extractionSize = extractionSize + uint64(hdr.Size)
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				return handleError(c, m, "max extraction size exceeded", err)
			}

			// create file
			if err := unpackTarget.CreateSafeFile(c, dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(c, m, "failed to create safe file", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store metrics
			m.ExtractionSize = int64(extractionSize)
			m.ExtractedFiles++
			continue

		// its a symlink !!
		case tar.TypeSymlink:

			// check if symlinks are allowed
			if !c.AllowSymlinks() {

				// check for continue for unsupported files
				if c.ContinueOnUnsupportedFiles() {
					m.SkippedUnsupportedFiles++
					m.LastSkippedUnsupportedFile = hdr.Name
					continue
				}

				if err := handleError(c, m, "symlinks are not allowed", fmt.Errorf("symlinks are not allowed")); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// create link
			if err := unpackTarget.CreateSafeSymlink(c, dst, hdr.Name, hdr.Linkname); err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(c, m, "failed to create safe symlink", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store metrics and continue
			m.ExtractedSymlinks++
			continue

		default:

			// check if unsupported files should be skipped
			if c.ContinueOnUnsupportedFiles() {
				m.SkippedUnsupportedFiles++
				m.LastSkippedUnsupportedFile = hdr.Name
				continue
			}

			// increase error counter, set error and end if necessary
			if err := handleError(c, m, "cannot extract file", fmt.Errorf("unsupported filetype in archive (%x)", hdr.Typeflag)); err != nil {
				return err
			}

			// do not end on error
			continue
		}
	}
}
