package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/ulikunitz/xz"
)

// magicBytesXz is the magic bytes for xz files.
// reference https://tukaani.org/xz/xz-file-format-1.0.4.txt
var magicBytesXz = [][]byte{
	{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
}

// fileExtensionXz is the file extension for xz files.
var fileExtensionXz = "xz"

// IsXz checks if the header matches the xz magic bytes.
func IsXz(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesXz)
}

// Unpack sets a timeout for the ctx and starts the xz decompression from src to dst.
func UnpackXz(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(c)

	// unpack
	return unpackXz(ctx, src, dst, c)
}

// Unpack decompresses src with xz algorithm into dst.
func unpackXz(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: fileExtensionXz}
	defer c.MetricsHook(ctx, &metrics)

	// prepare xz extraction
	c.Logger().Info("extracting xz")
	limitedReader := limitReader(src, c)
	xzStream, err := xz.NewReader(limitedReader)
	if err != nil {
		return handleError(c, &metrics, "cannot create xz reader", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, fmt.Sprintf(".%s", fileExtensionXz))

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, xzStream, 0644); err != nil {
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
