package extractor

import (
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
)

// magicBytesZlib is the magic bytes for Zlib files.
// reference https://www.ietf.org/rfc/rfc1950.txt
var magicBytesZlib = [][]byte{
	{0x78, 0x01},
	{0x78, 0x5e},
	{0x78, 0x9c},
	{0x78, 0xda},
	{0x78, 0x20},
	{0x78, 0x7d},
	{0x78, 0xbb},
	{0x78, 0xf9},
}

// fileExtensionZlib is the file extension for Zlib files.
var fileExtensionZlib = "zz"

// IsZlib checks if the header matches the Zlib magic bytes.
func IsZlib(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZlib)
}

// Unpack sets a timeout for the ctx and starts the zlib decompression from src to dst.
func UnpackZlib(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(c)

	// unpack
	return unpackZlib(ctx, src, dst, c)
}

// Unpack decompresses src with zlib algorithm into dst.
func unpackZlib(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: fileExtensionZlib}
	defer c.MetricsHook(ctx, &metrics)

	// prepare zlib decompression extraction
	c.Logger().Info("extracting zlib")
	limitedReader := limitReader(src, c)
	zlibStream, err := zlib.NewReader(limitedReader)
	if err != nil {
		return handleError(c, &metrics, "cannot create zlib reader", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, fmt.Sprintf(".%s", fileExtensionZlib))

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, zlibStream, 0644); err != nil {
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
