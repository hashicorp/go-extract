package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

type decompressionFunction func(io.Reader, *config.Config) (io.Reader, error)

func decompress(ctx context.Context, src io.Reader, dst string, c *config.Config, decom decompressionFunction, fileExt string) error {

	// prepare telemetry capturing
	// remark: do not defer TelemetryHook here, bc/ in case of tar.<compression>, the
	// tar extractor should submit the telemetry data
	c.Logger().Info("decompress", "fileExt", fileExt)
	td := &telemetry.Data{ExtractedType: fileExt}
	defer c.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// limit input size
	limitedReader := NewLimitErrorReader(src, c.MaxInputSize())
	defer captureInputSize(td, limitedReader)

	// start decompression
	decompressedStream, err := decom(limitedReader, c)
	if err != nil {
		return handleError(c, td, "cannot start decompression", err)
	}
	defer func() {
		if closer, ok := decompressedStream.(io.Closer); ok {
			closer.Close()
		}
	}()
	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, td, "context error", err)
	}

	// convert to peek header
	headerReader, err := NewHeaderReader(decompressedStream, MaxHeaderLength)
	if err != nil {
		return handleError(c, td, "cannot read uncompressed header", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, td, "context error", err)
	}

	// check if uncompressed stream is tar
	headerBytes := headerReader.PeekHeader()

	// check for tar header
	checkUntar := !c.NoUntarAfterDecompression()
	if checkUntar && IsTar(headerBytes) {
		td.ExtractedType = fmt.Sprintf("tar.%s", fileExt) // combine types
		return unpackTar(ctx, headerReader, dst, c, td)
	}

	// determine name and decompress content
	dst, outputName := determineOutputName(dst, src)
	c.Logger().Debug("determined output name", "name", outputName)

	// check if dst needs to be created
	if c.CreateDestination() {
		if err := createDir(c, dst, ".", c.DefaultDirPermission()); err != nil {
			return handleError(c, td, "cannot create destination", err)
		}
	}

	// check if dst exist
	if _, err := os.Stat(dst); err != nil {
		return handleError(c, td, "destination does not exist", err)
	}

	// decompress content and write to file
	if _, err := createFile(c, dst, outputName, headerReader, c.DefaultFilePermission(), c.MaxExtractionSize()); err != nil {
		return handleError(c, td, "cannot create file", err)
	}

	// capture telemetry
	stat, err := os.Stat(filepath.Join(dst, outputName))
	if err != nil {
		return handleError(c, td, "cannot stat file", err)
	}
	td.ExtractionSize = stat.Size()
	td.ExtractedFiles++

	// finished
	return nil

}
