package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/golang/snappy"
	"github.com/hashicorp/go-extract/config"
)

// magicBytesSnappy is the magic bytes for snappy files.
var magicBytesSnappy = [][]byte{
	append([]byte{0xff, 0x06, 0x00, 0x00}, []byte("sNaPpY")...),
}

// fileExtensionSnappy is the file extension for snappy files.
var fileExtensionSnappy = "sz"

// IsSnappy checks if the header matches the snappy magic bytes.
func IsSnappy(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesSnappy)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func UnpackSnappy(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(ctx, c)

	// unpack
	return unpackSnappy(ctx, src, dst, c)
}

// Unpack decompresses src with snappy algorithm into dst.
func unpackSnappy(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: fileExtensionSnappy}
	defer c.MetricsHook(ctx, &metrics)

	// prepare gzip extraction
	c.Logger().Info("extracting snappy")
	limitedReader := limitReader(ctx, src, c)
	brotliStream := snappy.NewReader(limitedReader)

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, fmt.Sprintf(".%s", fileExtensionSnappy))

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, brotliStream, 0644); err != nil {
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
