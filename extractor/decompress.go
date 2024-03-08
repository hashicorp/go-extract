package extractor

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
)

type decompressionFunction func(io.Reader, *config.Config) (io.Reader, error)

func decompress(ctx context.Context, src io.Reader, dst string, c *config.Config, decom decompressionFunction, fileExt string) error {

	// prepare telemetry capturing
	c.Logger().Info("decompress", "fileExt", fileExt)
	captureExtractionDuration(c)
	metrics := config.Metrics{ExtractedType: fileExt}
	defer c.MetricsHook(ctx, &metrics)

	// prepare decompression
	limitedReader := limitReader(src, c)
	decompressedStream, err := decom(limitedReader, c)
	if err != nil {
		return handleError(c, &metrics, "cannot start decompression", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// determine name and decompress content
	dst, outputName := determineOutputName(dst, src)
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, decompressedStream, 0644); err != nil {
		return handleError(c, &metrics, "cannot create file", err)
	}

	// capture telemetry
	if stat, err := os.Stat(filepath.Join(dst, outputName)); err == nil {
		metrics.ExtractionSize = stat.Size()
	}
	metrics.ExtractedFiles++

	// finished
	return nil

}
