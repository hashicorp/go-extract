package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
)

type decompressionFunction func(io.Reader, *config.Config) (io.Reader, error)

func decompress(ctx context.Context, src io.Reader, dst string, c *config.Config, decom decompressionFunction, fileExt string) error {

	// prepare telemetry capturing
	// remark: do not defer MetricsHook here, bc/ in case of tar.<compression>, the
	// tar extractor should submit the metrics
	c.Logger().Info("decompress", "fileExt", fileExt)
	captureExtractionDuration(c)
	metrics := config.Metrics{ExtractedType: fileExt}

	// prepare decompression
	limitedReader := limitReader(src, c)
	decompressedStream, err := decom(limitedReader, c)
	if err != nil {
		defer c.MetricsHook(ctx, &metrics)
		return handleError(c, &metrics, "cannot start decompression", err)
	}
	defer func() {
		if closer, ok := decompressedStream.(io.Closer); ok {
			closer.Close()
		}
	}()
	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// convert to peek header
	headerReader, err := NewHeaderReader(decompressedStream, MaxHeaderLength)
	if err != nil {
		defer c.MetricsHook(ctx, &metrics)
		return handleError(c, &metrics, "cannot read uncompressed header", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, &metrics, "context error", err)
	}

	// check if uncompressed stream is tar
	headerBytes := headerReader.PeekHeader()

	// check for tar header
	checkUntar := !c.NoUntarAfterDecompression()
	if checkUntar && IsTar(headerBytes) {
		// combine types
		c.AddMetricsProcessor(func(ctx context.Context, m *config.Metrics) {
			m.ExtractedType = fmt.Sprintf("%s.%s", m.ExtractedType, fileExt)
		})

		// continue with tar extraction
		return UnpackTar(ctx, headerReader, dst, c)
	}

	// ensure metrics are emitted
	defer c.MetricsHook(ctx, &metrics)

	// determine name and decompress content
	dst, outputName := determineOutputName(dst, src)
	c.Logger().Debug("determined output name", "name", outputName)
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, headerReader, 0644); err != nil {
		return handleError(c, &metrics, "cannot create file", err)
	}

	// capture telemetry
	stat, err := os.Stat(filepath.Join(dst, outputName))
	if err != nil {
		return handleError(c, &metrics, "cannot stat file", err)
	}
	metrics.ExtractionSize = stat.Size()
	metrics.ExtractedFiles++

	// finished
	return nil

}
