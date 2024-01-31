package extractor

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// reference https://socketloop.com/tutorials/golang-gunzip-file

var magicBytesGZIP = [][]byte{
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

type HeaderCheck (func([]byte) bool)

func IsGZIP(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZIP)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func (g *Gzip) Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {

	// prepare limits input and ensures metrics capturing
	reader := prepare(ctx, src, c)

	return g.unpack(ctx, reader, dst, t, c)
}

// Unpack decompresses src with gzip algorithm into dst. If src is a gziped tar archive,
// the tar archive is extracted
func (gz *Gzip) unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {

	// object to store metrics
	metrics := config.Metrics{ExtractedType: "gzip"}

	// emit metrics
	defer c.MetricsHooksOnce(ctx, &metrics)

	// prepare gzip extraction
	c.Logger().Info("extracting gzip")
	uncompressedStream, err := gzip.NewReader(src)
	if err != nil {
		msg := "cannot read gzip"
		return handleError(c, &metrics, msg, err)
	}

	// convert to peek header
	headerReader, err := NewHeaderReader(uncompressedStream, MaxHeaderLength)
	if err != nil {
		msg := "cannot read header uncompressed gzip"
		return handleError(c, &metrics, msg, err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		msg := "context error"
		return handleError(c, &metrics, msg, err)
	}

	// check if uncompressed stream is tar
	headerBytes := headerReader.PeekHeader()

	// check for tar header
	if c.TarGzExtract() && IsTar(headerBytes) {
		// combine types
		c.AddMetricsHook(func(ctx context.Context, m *config.Metrics) {
			m.ExtractedType = fmt.Sprintf("%s+gzip", m.ExtractedType)
		})

		// continue with tar extraction
		return NewTar().Unpack(ctx, headerReader, dst, t, c)
	}

	// determine name for decompressed content
	// TODO: use headerReader to determine name
	name := "gunziped-content"
	if dst != "." {
		if stat, err := os.Stat(dst); os.IsNotExist(err) || stat.Mode()&fs.ModeDir == 0 {
			name = filepath.Base(dst)
			dst = filepath.Dir(dst)
		}
	}

	// Create file
	if err := t.CreateSafeFile(c, dst, name, headerReader, 0644); err != nil {
		msg := "cannot create file"
		return handleError(c, &metrics, msg, err)
	}

	// get size of extracted file
	if stat, err := os.Stat(filepath.Join(dst, name)); err == nil {
		metrics.ExtractionSize = stat.Size()
	}

	// finished
	metrics.ExtractedFiles++
	return nil
}
