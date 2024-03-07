package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/klauspost/compress/zstd"
)

// magicBytesZstd is the magic bytes for zstandard files.
// reference: https://www.rfc-editor.org/rfc/rfc8878.html
var magicBytesZstd = [][]byte{
	{0x28, 0xb5, 0x2f, 0xfd},
}

// fileExtensionZstd is the file extension for zstandard files.
var fileExtensionZstd = "zst"

// IsZstd checks if the header matches the zstandard magic bytes.
func IsZstd(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZstd)
}

// Unpack sets a timeout for the ctx and starts the zstandard decompression from src to dst.
func UnpackZstd(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(ctx, c)

	// unpack
	return unpackZstd(ctx, src, dst, c)
}

// Unpack decompresses src with zstandard algorithm into dst.
func unpackZstd(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: fileExtensionZstd}
	defer c.MetricsHook(ctx, &metrics)

	// prepare extraction
	c.Logger().Info("extracting zstd")
	limitedReader := limitReader(ctx, src, c)
	zstandardDecoder, err := zstd.NewReader(limitedReader)
	if err != nil {
		return handleError(c, &metrics, "cannot create zstd decoder", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, fmt.Sprintf(".%s", fileExtensionZstd))

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, zstandardDecoder, 0640); err != nil {
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
