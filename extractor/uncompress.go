package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/metrics"
)

type uncompressionFunction func(io.Reader, *config.Config) (io.Reader, error)

func uncompress(ctx context.Context, src io.Reader, dst string, c *config.Config, uncompressor uncompressionFunction, fileExt string) error {

	// prepare telemetry capturing
	// remark: do not defer MetricsHook here, bc/ in case of tar.<compression>, the
	// tar extractor should submit the metrics
	c.Logger().Info("uncompress", "fileExt", fileExt)
	m := &metrics.Metrics{ExtractedType: fileExt}
	defer c.MetricsHook()(ctx, m)
	defer captureExtractionDuration(m, now())

	// prepare uncompression
	limitedReader := NewLimitErrorReader(src, c.MaxInputSize())
	defer captureInputSize(m, limitedReader)
	uncompressedStream, err := uncompressor(limitedReader, c)
	if err != nil {
		return handleError(c, m, "cannot start uncompression", err)
	}
	defer func() {
		if closer, ok := uncompressedStream.(io.Closer); ok {
			closer.Close()
		}
	}()
	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, m, "context error", err)
	}

	// convert to peek header
	headerReader, err := NewHeaderReader(uncompressedStream, MaxHeaderLength)
	if err != nil {
		return handleError(c, m, "cannot read uncompressed header", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, m, "context error", err)
	}

	// check if uncompressed stream is tar
	headerBytes := headerReader.PeekHeader()

	// check for tar header
	checkUnTar := !c.NoUntarAfterUncompression()
	if checkUnTar && IsTar(headerBytes) {
		m.ExtractedType = fmt.Sprintf("tar.%s", fileExt)
		return unpackTar(ctx, headerReader, dst, c, m)
	}

	// determine name and uncompress content
	dst, outputName := determineOutputName(dst, src)
	c.Logger().Debug("determined output name", "name", outputName)
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, headerReader, 0644); err != nil {
		return handleError(c, m, "cannot create file", err)
	}

	// capture telemetry
	stat, err := os.Stat(filepath.Join(dst, outputName))
	if err != nil {
		return handleError(c, m, "cannot stat file", err)
	}
	m.ExtractionSize = stat.Size()
	m.ExtractedFiles++

	// finished
	return nil

}
