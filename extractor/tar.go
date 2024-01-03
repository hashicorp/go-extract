package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

const OffsetTar = 257

var MagicBytesTar = [][]byte{
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x20, 0x00},
}

// Tar holds information that are needed for tar extraction.
type Tar struct {
	// target is the target of the extraction
	target target.Target
}

// NewTar creates a new untar object with config as configuration
func NewTar() *Tar {
	// configure target
	target := target.NewOs()

	// instantiate
	tar := Tar{
		target: target,
	}

	// return the modified house instance
	return &tar
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func (t *Tar) Unpack(ctx context.Context, src io.Reader, dst string, target target.Target, c *config.Config) error {
	return t.unpack(ctx, src, dst, target, c)
}

// unpack checks ctx for cancellation, while it reads a tar file from src and extracts the contents to dst.
func (t *Tar) unpack(ctx context.Context, src io.Reader, dst string, target target.Target, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{}
	metrics.ExtractedType = "tar"
	start := time.Now()

	// anonymous function to emit metrics
	defer func() {

		// calculate execution time
		metrics.ExtractionDuration = time.Since(start)

		// emit metrics
		if c.MetricsHook != nil {
			c.MetricsHook(metrics)
		}
	}()

	// prepare safety vars
	var objectCounter int64
	var extractionSize uint64

	// open tar
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
			return processError(c, &metrics, msg, err)

		// if the header is nil, just skip it (not sure how this happens)
		case hdr == nil:
			continue
		}

		// check for to many objects in archive
		objectCounter++

		// check if maximum of objects is exceeded
		if err := c.CheckMaxObjects(objectCounter); err != nil {
			msg := "max objects check failed"
			return processError(c, &metrics, msg, err)
		}

		// check if name is just current working dir
		if filepath.Clean(hdr.Name) == "." {
			continue
		}

		c.Log.Debug("extract", "name", hdr.Name)
		switch hdr.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:

			// handle directory
			if err := t.target.CreateSafeDir(c, dst, hdr.Name); err != nil {
				msg := "failed to create safe directory"
				if err := processError(c, &metrics, msg, err); err != nil {
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
				return processError(c, &metrics, msg, err)
			}

			// create file
			if err := t.target.CreateSafeFile(c, dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {

				// increase error counter, set error and end if necessary
				msg := "failed to create safe file"
				if err := processError(c, &metrics, msg, err); err != nil {
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

			// create link
			if err := t.target.CreateSafeSymlink(c, dst, hdr.Name, hdr.Linkname); err != nil {

				// increase error counter, set error and end if necessary
				msg := "failed to create safe symlink"
				if err := processError(c, &metrics, msg, err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store metrics and continue
			metrics.ExtractedSymlinks++
			continue

		default:

			// increase error counter, set error and end if necessary
			err := fmt.Errorf("unsupported filetype in archive (%x)", hdr.Typeflag)
			msg := "cannot extract file"
			if err := processError(c, &metrics, msg, err); err != nil {
				return err
			}

			// do not end on error
			continue
		}
	}
}
