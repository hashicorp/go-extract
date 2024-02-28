package extractor

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/andybalholm/brotli"
	"github.com/hashicorp/go-extract/config"
)

// magicBytesBrotli are the magic bytes for brotli compressed files
// reference https://github.com/madler/brotli/blob/1d428d3a9baade233ebc3ac108293256bcb813d1/br-format-v3.txt#L114-L116
var magicBytesBrotli = [][]byte{
	{0xce, 0xb2, 0xcf, 0x81},
}

// IsBrotli checks if the header matches the magic bytes for brotli compressed files
func IsBrotli(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesBrotli)
}

// Unpack sets a timeout for the ctx and starts the brotli decompression from src to dst.
func UnpackBrotli(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(ctx, c)

	// unpack
	return unpackBrotli(ctx, src, dst, c)
}

// Unpack decompresses src with brotli algorithm into dst.
func unpackBrotli(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: "brotli"}
	defer c.MetricsHook(ctx, &metrics)

	// prepare gzip extraction
	c.Logger().Info("extracting brotli")
	limitedReader := limitReader(ctx, src, c)
	brotliStream := brotli.NewReader(limitedReader)

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, ".br")

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, brotliStream, 0640); err != nil {
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
