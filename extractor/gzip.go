package extractor

import (
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
)

// reference https://socketloop.com/tutorials/golang-gunzip-file

var magicBytesGZip = [][]byte{
	{0x1f, 0x8b},
}

func IsGZip(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZip)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func UnpackGZip(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// prepare limits input and ensures metrics capturing
	reader := prepare(ctx, src, c)

	return unpackGZip(ctx, reader, dst, c)
}

// Unpack decompresses src with gzip algorithm into dst. If src is a gziped tar archive,
// the tar archive is extracted
func unpackGZip(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	// remark: do not setup MetricsHook here, bc/ in case of tar+gzip, the
	// tar extractor should submit the metrics
	metrics := config.Metrics{ExtractedType: "gzip"}

	// prepare gzip extraction
	c.Logger().Info("extracting gzip")
	uncompressedStream, err := gzip.NewReader(src)
	if err != nil {
		defer c.MetricsHook(ctx, &metrics)
		return handleError(c, &metrics, "cannot read gzip", err)
	}
	defer uncompressedStream.Close()

	// convert to peek header
	headerReader, err := NewHeaderReader(uncompressedStream, MaxHeaderLength)
	if err != nil {
		defer c.MetricsHook(ctx, &metrics)
		return handleError(c, &metrics, "cannot read header uncompressed gzip", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		defer c.MetricsHook(ctx, &metrics)
		return handleError(c, &metrics, "context error", err)
	}

	// check if uncompressed stream is tar
	headerBytes := headerReader.PeekHeader()

	// check for tar header
	if c.UntarAfterDecompression() && IsTar(headerBytes) {
		// combine types
		c.AddMetricsProcessor(func(ctx context.Context, m *config.Metrics) {
			m.ExtractedType = "tar+gzip"
		})

		// continue with tar extraction
		return UnpackTar(ctx, headerReader, dst, c)
	}

	// ensure metrics are emitted
	defer c.MetricsHook(ctx, &metrics)

	// determine name for decompressed content
	// TODO: use headerReader to determine name
	name := "gunziped-content"
	if dst != "." {
		if stat, err := os.Stat(dst); os.IsNotExist(err) || stat.Mode()&fs.ModeDir == 0 {
			name = filepath.Base(dst)
			dst = filepath.Dir(dst)
		}
	}

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, name, headerReader, 0644); err != nil {
		return handleError(c, &metrics, "cannot create file", err)
	}

	// get size of extracted file
	if stat, err := os.Stat(filepath.Join(dst, name)); err == nil {
		metrics.ExtractionSize = stat.Size()
	}

	// finished
	metrics.ExtractedFiles++
	return nil
}
