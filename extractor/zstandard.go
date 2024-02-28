package extractor

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/klauspost/compress/zstd"
)

// magicBytesZstandard is the magic bytes for zstandard files.
// reference: https://www.rfc-editor.org/rfc/rfc8878.html
var magicBytesZstandard = [][]byte{
	{0x28, 0xb5, 0x2f, 0xfd},
}

// fileExtensionZstandard is the file extension for zstandard files.
var fileExtensionZstandard = "zst"

// IsZstandard checks if the header matches the zstandard magic bytes.
func IsZstandard(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZstandard)
}

// Unpack sets a timeout for the ctx and starts the zstandard decompression from src to dst.
func UnpackZstandard(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(ctx, c)

	// unpack
	return unpackZstandard(ctx, src, dst, c)
}

// Unpack decompresses src with zstandard algorithm into dst.
func unpackZstandard(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: fileExtensionZstandard}
	defer c.MetricsHook(ctx, &metrics)

	// prepare extraction
	c.Logger().Info("extracting zstandard")
	limitedReader := limitReader(ctx, src, c)
	zstandardDecoder, err := zstd.NewReader(limitedReader)
	if err != nil {
		return handleError(c, &metrics, "cannot create zstandard decoder", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, fileExtensionZstandard)

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
