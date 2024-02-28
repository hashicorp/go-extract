package extractor

import (
	"compress/bzip2"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
)

// magicBytesBzip2 are the magic bytes for bzip2 compressed files
// reference: https://en.wikipedia.org/wiki/Bzip2 // https://github.com/dsnet/compress/blob/master/doc/bzip2-format.pdf
var magicBytesBzip2 = [][]byte{
	[]byte("BZh1"),
	[]byte("BZh2"),
	[]byte("BZh3"),
	[]byte("BZh4"),
	[]byte("BZh5"),
	[]byte("BZh6"),
	[]byte("BZh7"),
	[]byte("BZh8"),
	[]byte("BZh9"),
}

// IsBzip2 checks if the header matches the magic bytes for bzip2 compressed files
func IsBzip2(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesBzip2)
}

// Unpack sets a timeout for the ctx and starts the bzip2 decompression from src to dst.
func UnpackBzip2(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// capture extraction duration
	captureExtractionDuration(ctx, c)

	// unpack
	return unpackBzip2(ctx, src, dst, c)
}

// Unpack decompresses src with bzip2 algorithm into dst.
func unpackBzip2(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: "bzip2"}
	defer c.MetricsHook(ctx, &metrics)

	// prepare bzip2 extraction
	c.Logger().Info("extracting bzip2")
	limitedReader := limitReader(ctx, src, c)
	bzip2Stream := bzip2.NewReader(limitedReader)

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name for decompressed content
	dst, outputName := determineOutputName(dst, src, ".bz2")

	// Create file
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, bzip2Stream, 0644); err != nil {
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
