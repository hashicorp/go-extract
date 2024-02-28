package extractor

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
)

// magicBytesGZip are the magic bytes for gzip compressed files
// reference https://socketloop.com/tutorials/golang-gunzip-file
var magicBytesGZip = [][]byte{
	{0x1f, 0x8b},
}

// IsGZip checks if the header matches the magic bytes for gzip compressed files
func IsGZip(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZip)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func UnpackGZip(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(ctx, c)

	return unpackGZip(ctx, src, dst, c)
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
	limitedReader := limitReader(ctx, src, c)
	gunzipedStream, err := gzip.NewReader(limitedReader)
	if err != nil {
		defer c.MetricsHook(ctx, &metrics)
		return handleError(c, &metrics, "cannot read gzip", err)
	}
	defer gunzipedStream.Close()

	// convert to peek header
	headerReader, err := NewHeaderReader(gunzipedStream, MaxHeaderLength)
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
	checkUntar := !c.NoUntarAfterDecompression()
	if checkUntar && IsTar(headerBytes) {
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
	dst, outputName := determineOutputName(dst, src, ".gz")

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, headerReader, 0644); err != nil {
		return handleError(c, &metrics, "cannot create file", err)
	}

	// get size of extracted file
	if stat, err := os.Stat(filepath.Join(dst, outputName)); err == nil {
		metrics.ExtractionSize = stat.Size()
	}

	// finished
	metrics.ExtractedFiles++
	return nil
}
