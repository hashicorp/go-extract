package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/pierrec/lz4/v4"
)

// magicBytesLZ4 is the magic bytes for LZ4 files.
// reference https://android.googlesource.com/platform/external/lz4/+/HEAD/doc/lz4_Frame_format.md
var magicBytesLZ4 = [][]byte{
	{0x18, 0x4D, 0x22, 0x04},
}

// fileExtensionLZ4 is the file extension for LZ4 files.
var fileExtensionLZ4 = "lz4"

// IsLZ4 checks if the header matches the LZ4 magic bytes.
func IsLZ4(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesLZ4)
}

// Unpack sets a timeout for the ctx and starts the lz4 decompression from src to dst.
func UnpackLZ4(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(c)

	// unpack
	return unpackLZ4(ctx, src, dst, c)
}

// Unpack decompresses src with lz4 algorithm into dst.
func unpackLZ4(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: fileExtensionLZ4}
	defer c.MetricsHook(ctx, &metrics)

	// prepare lz4 decompression extraction
	c.Logger().Info("extracting lz4")
	limitedReader := limitReader(src, c)
	lz4Stream := lz4.NewReader(limitedReader)

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, fmt.Sprintf(".%s", fileExtensionLZ4))

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, lz4Stream, 0644); err != nil {
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
