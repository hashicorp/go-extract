package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

// offsetTar is the offset where the magic bytes are located in the file
const offsetTar = 257

// fileExtensionTar is the file extension for tar files
var fileExtensionTar = "tar"

// magicBytesTar are the magic bytes for tar files
var magicBytesTar = [][]byte{
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x20, 0x00},
}

// IsTar checks if the header matches the magic bytes for tar files
func IsTar(data []byte) bool {
	return matchesMagicBytes(data, offsetTar, magicBytesTar)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func UnpackTar(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// prepare telemetry capturing
	m := &telemetry.Data{ExtractedType: fileExtensionTar}
	defer c.TelemetryHook()(ctx, m)
	defer captureExtractionDuration(m, now())

	// prepare reader
	limitedReader := NewLimitErrorReader(src, c.MaxInputSize())
	defer captureInputSize(m, limitedReader)

	// start extraction
	return unpackTar(ctx, limitedReader, dst, c, m)
}

// unpack checks ctx for cancellation, while it reads a tar file from src and extracts the contents to dst.
func unpackTar(ctx context.Context, src io.Reader, dst string, c *config.Config, td *telemetry.Data) error {

	// start extraction
	c.Logger().Info("extracting tar")
	tr := tar.NewReader(src)
	var objectCounter int64
	var extractionSize uint64
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
			return handleError(c, td, "error reading tar", err)

		// if the header is nil, just skip it (not sure how this happens)
		case hdr == nil:
			continue
		}

		// check for to many objects in archive
		objectCounter++

		// check if maximum of objects is exceeded
		if err := c.CheckMaxObjects(objectCounter); err != nil {
			return handleError(c, td, "max objects check failed", err)
		}

		// check if file needs to match patterns
		match, err := checkPatterns(c.Patterns(), hdr.Name)
		if err != nil {
			return handleError(c, td, "cannot check pattern", err)
		}
		if !match {
			c.Logger().Info("skipping file (pattern mismatch)", "name", hdr.Name)
			td.PatternMismatches++
			continue
		}

		c.Logger().Debug("extract", "name", hdr.Name)
		switch hdr.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:

			// handle directory
			if err := unpackTarget.CreateSafeDir(c, dst, hdr.Name); err != nil {
				if err := handleError(c, td, "failed to create safe directory", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store telemetry and continue
			td.ExtractedDirs++
			continue

		// if it's a file create it
		case tar.TypeReg:

			// check extraction size
			extractionSize = extractionSize + uint64(hdr.Size)
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				return handleError(c, td, "max extraction size exceeded", err)
			}

			// create file
			if err := unpackTarget.CreateSafeFile(c, dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(c, td, "failed to create safe file", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store telemetry
			td.ExtractionSize = int64(extractionSize)
			td.ExtractedFiles++
			continue

		// its a symlink !!
		case tar.TypeSymlink:

			// check if symlinks are allowed
			if c.DenySymlinkExtraction() {

				// check for continue for unsupported files
				if c.ContinueOnUnsupportedFiles() {
					td.UnsupportedFiles++
					td.LastUnsupportedFile = hdr.Name
					continue
				}

				if err := handleError(c, td, "symlinks are not allowed", fmt.Errorf("symlinks are not allowed")); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// create link
			if err := unpackTarget.CreateSafeSymlink(c, dst, hdr.Name, hdr.Linkname); err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(c, td, "failed to create safe symlink", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store telemetry and continue
			td.ExtractedSymlinks++
			continue

		default:

			// check for git comment file `pax_global_header` from type `67` and skip
			if hdr.Typeflag&tar.TypeXGlobalHeader == tar.TypeXGlobalHeader && hdr.Name == "pax_global_header" {
				continue
			}

			// check if unsupported files should be skipped
			if c.ContinueOnUnsupportedFiles() {
				td.UnsupportedFiles++
				td.LastUnsupportedFile = hdr.Name
				continue
			}

			// increase error counter, set error and end if necessary
			if err := handleError(c, td, "cannot extract file", fmt.Errorf("unsupported filetype in archive (%x)", hdr.Typeflag)); err != nil {
				return err
			}

			// do not end on error
			continue
		}
	}
}
