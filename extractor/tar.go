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
			metrics.ExtractionErrors++
			metrics.LastExtractionError = err
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case hdr == nil:
			continue
		}

		// check for to many objects in archive
		objectCounter++

		// check if maximum of objects is exceeded
		if err := c.CheckMaxObjects(objectCounter); err != nil {
			metrics.ExtractionErrors++
			metrics.LastExtractionError = err
			return err
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

				// increase error counter and set error
				metrics.ExtractionErrors++
				metrics.LastExtractionError = err

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe directory", "error", err)
					continue
				}

				// end extraction
				return err
			}
			metrics.ExtractedDirs++
			continue

		// if it's a file create it
		case tar.TypeReg:

			// check extraction size
			extractionSize = extractionSize + uint64(hdr.Size)
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				metrics.ExtractionErrors++
				metrics.LastExtractionError = err
				return err
			}

			if err := t.target.CreateSafeFile(c, dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {

				// increase error counter and set error
				metrics.ExtractionErrors++
				metrics.LastExtractionError = err

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe file", "error", err)
					continue
				}

				// end extraction
				return err
			}
			metrics.ExtractionSize = int64(extractionSize)
			metrics.ExtractedFiles++

		// its a symlink !!
		case tar.TypeSymlink:
			// create link
			if err := t.target.CreateSafeSymlink(c, dst, hdr.Name, hdr.Linkname); err != nil {

				// increase error counter and set error
				metrics.ExtractionErrors++
				metrics.LastExtractionError = err

				// do not end on error
				if c.ContinueOnError {
					c.Log.Debug("failed to create safe symlink", "error", err)
					continue
				}

				// end extraction
				return err
			}
			metrics.ExtractedSymlinks++

		default:
			metrics.ExtractionErrors++
			metrics.LastExtractionError = fmt.Errorf("unsupported filetype in archive")
			return metrics.LastExtractionError
		}
	}
}
