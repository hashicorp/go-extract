package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
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

// NewTar creates a new untar object with config as configuration
func NewTar() *Tar {

	// instantiate
	tar := Tar{}

	// return the modified house instance
	return &tar
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func (t *Tar) Unpack(ctx context.Context, src io.Reader, dst string, target target.Target, c *config.Config) error {

	// prepare limits input and ensures metrics capturing
	reader := prepare(ctx, src, c)

	return t.unpack(ctx, reader, dst, target, c)
}

// unpack checks ctx for cancellation, while it reads a tar file from src and extracts the contents to dst.
func (t *Tar) unpack(ctx context.Context, src io.Reader, dst string, target target.Target, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: "tar"}

	// anonymous function to emit metrics
	defer c.MetricsHook(ctx, &metrics)

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

		// if no more files are found return
		case err == io.EOF:
			// reached end of archive
			return nil

		// return any other error
		case err != nil:
			msg := "error reading tar"
			return handleError(c, &metrics, msg, err)

		// if the header is nil, just skip it (not sure how this happens)
		case hdr == nil:
			continue
		}

		// check for to many objects in archive
		objectCounter++

		// check if maximum of objects is exceeded
		if err := c.CheckMaxObjects(objectCounter); err != nil {
			msg := "max objects check failed"
			return handleError(c, &metrics, msg, err)
		}

		// check if name is just current working dir
		if filepath.Clean(hdr.Name) == "." {
			continue
		}

		c.Logger().Debug("extract", "name", hdr.Name)
		switch hdr.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:

			// handle directory
			if err := target.CreateSafeDir(c, dst, hdr.Name); err != nil {
				msg := "failed to create safe directory"
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store metrics and continue
			metrics.ExtractedDirs++
			continue

		// if it's a file create it
		case tar.TypeReg:

			// check extraction size
			extractionSize = extractionSize + uint64(hdr.Size)
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				msg := "max extraction size exceeded"
				return handleError(c, &metrics, msg, err)
			}

			// create file
			if err := target.CreateSafeFile(c, dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {

				// increase error counter, set error and end if necessary
				msg := "failed to create safe file"
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store metrics
			metrics.ExtractionSize = int64(extractionSize)
			metrics.ExtractedFiles++
			continue

		// its a symlink !!
		case tar.TypeSymlink:

			// check if symlinks are allowed
			if !c.AllowSymlinks() {

				// check for continue for unsupported files
				if c.ContinueOnUnsupportedFiles() {
					metrics.SkippedUnsupportedFiles++
					metrics.LastSkippedUnsupportedFile = hdr.Name
					continue
				}

				msg := "symlinks are not allowed"
				err := fmt.Errorf("symlinks are not allowed")
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// create link
			if err := target.CreateSafeSymlink(c, dst, hdr.Name, hdr.Linkname); err != nil {

				// increase error counter, set error and end if necessary
				msg := "failed to create safe symlink"
				if err := handleError(c, &metrics, msg, err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store metrics and continue
			metrics.ExtractedSymlinks++
			continue

		default:

			// check if unsupported files should be skipped
			if c.ContinueOnUnsupportedFiles() {
				metrics.SkippedUnsupportedFiles++
				metrics.LastSkippedUnsupportedFile = hdr.Name
				continue
			}

			// increase error counter, set error and end if necessary
			err := fmt.Errorf("unsupported filetype in archive (%x)", hdr.Typeflag)
			msg := "cannot extract file"
			if err := handleError(c, &metrics, msg, err); err != nil {
				return err
			}

			// do not end on error
			continue
		}
	}
}
