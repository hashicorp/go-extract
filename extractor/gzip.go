package extractor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// reference https://socketloop.com/tutorials/golang-gunzip-file

var MagicBytesGZIP = [][]byte{
	{0x1f, 0x8b},
}

// Gzip is a struct type that holds all information to perform an gzip decompression
type Gzip struct{}

// NewGzip returns a new Gzip object with config as configuration.
func NewGzip() *Gzip {
	// instantiate
	gzip := Gzip{}

	// return the modified house instance
	return &gzip
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func (g *Gzip) Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
	return g.unpack(ctx, src, dst, t, c)
}

// Unpack decompresses src with gzip algorithm into dst. If src is a gziped tar archive,
// the tar archive is extracted
func (gz *Gzip) unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {

	// ensure input size and capture metrics
	ler := NewLimitErrorReader(src, c.MaxInputSize)
	src = ler

	// object to store metrics
	metrics := config.Metrics{}
	metrics.ExtractedType = "gzip"
	start := time.Now()
	emitGzipMetrics := true

	// anonymous function to emit metrics
	emitMetrics := func() {

		// check if metrics should still be emitted
		if emitGzipMetrics {

			// store input file size
			metrics.InputSize = ler.N

			// calculate execution time
			metrics.ExtractionDuration = time.Since(start)

			// emit metrics
			if c.MetricsHook != nil {
				c.MetricsHook(ctx, metrics)
			}

		}
	}

	// emit metrics
	defer emitMetrics()

	c.Logger.Info("extracting gzip")

	uncompressedStream, err := gzip.NewReader(src)
	if err != nil {
		msg := "cannot read gzip"
		return handleError(c, &metrics, msg, err)
	}

	// size check
	var bytesBuffer bytes.Buffer
	if c.MaxExtractionSize > -1 {
		var readBytes int64
		for {
			buf := make([]byte, 1024)
			n, err := uncompressedStream.Read(buf)
			if err != nil && err != io.EOF {
				msg := "cannot read decompressed gzip"
				return handleError(c, &metrics, msg, err)
			}

			// clothing read
			if n == 0 {
				break
			}

			// check if maximum is exceeded
			if readBytes+int64(n) <= c.MaxExtractionSize {
				bytesBuffer.Write(buf[:n])
				readBytes = readBytes + int64(n)
				metrics.ExtractionSize = readBytes

				// check if context is canceled
				if ctx.Err() != nil {
					return nil
				}
			} else {
				err := fmt.Errorf("maximum extraction size exceeded")
				msg := "cannot continue decompress gzip"
				return handleError(c, &metrics, msg, err)
			}
		}
	} else {
		metrics.ExtractionSize, err = bytesBuffer.ReadFrom(uncompressedStream)
		if err != nil {
			msg := "cannot read from gzip"
			return handleError(c, &metrics, msg, err)
		}
	}

	// check if src is a tar archive
	c.Logger.Debug("check magic bytes")
	for _, magicBytes := range MagicBytesTar {

		// get decompressed gzip data
		data := bytesBuffer.Bytes()

		// skip if smaller than offset
		if OffsetTar+len(magicBytes) > len(data) {
			continue
		}

		// check if magic bytes match
		if bytes.Equal(magicBytes, data[OffsetTar:OffsetTar+len(magicBytes)]) {

			// ensure that gzip metrics are not emitted and tar metrics are combined with gzip metrics
			if c.MetricsHook != nil {
				emitGzipMetrics = false
				oldMetricsHook := c.MetricsHook
				c.MetricsHook = func(ctx context.Context, m config.Metrics) {
					m.ExtractedType = "tar+gzip"             // combined input type
					m.InputSize = int64(ler.ReadBytes())     // store original input file size
					m.ExtractionDuration = time.Since(start) // calculate execution time beginning from gzip start
					// emit metrics
					if oldMetricsHook != nil {
						oldMetricsHook(ctx, m)
					}
				}
			}

			// check if context is canceled
			if err := ctx.Err(); err != nil {
				msg := "context error"
				return handleError(c, &metrics, msg, err)
			}

			// extract tar archive
			tar := NewTar()
			return tar.Unpack(ctx, bytes.NewReader(data), dst, t, c)
		}
	}

	// determine name for decompressed content
	name := "gunziped-content"
	if dst != "." {
		if stat, err := os.Stat(dst); os.IsNotExist(err) || stat.Mode()&fs.ModeDir == 0 {
			name = filepath.Base(dst)
			dst = filepath.Dir(dst)
		}
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		msg := "context error"
		return handleError(c, &metrics, msg, err)
	}

	// Create file
	if err := t.CreateSafeFile(c, dst, name, bytes.NewReader(bytesBuffer.Bytes()), 0644); err != nil {
		msg := "cannot create file"
		return handleError(c, &metrics, msg, err)
	}

	// finished
	metrics.ExtractedFiles++
	return nil
}
