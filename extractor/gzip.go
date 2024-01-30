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

func matchesMagicBytes(data []byte, offset int, magicBytes [][]byte) bool {
	// check all possible magic bytes until match is found
	for _, mb := range magicBytes {
		// check if header is long enough
		if offset+len(mb) > len(data) {
			continue
		}

		// check for byte match
		if bytes.Equal(mb, data[offset:offset+len(mb)]) {
			return true
		}
	}

	// no match found
	return false
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

// HeaderReader is an implementation of io.Reader that allows the first bytes of
// the reader to be read twice. This is useful for identifying the archive type
// before unpacking.
type HeaderReader struct {
	r      io.Reader
	header []byte
}

func NewHeaderReader(r io.Reader, headerSize int) (*HeaderReader, error) {
	// read at least headerSize bytes. If EOF, capture whatever was read.
	buf := make([]byte, headerSize)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return &HeaderReader{r, buf[:n]}, nil
}

func (p *HeaderReader) Read(b []byte) (int, error) {
	// read from header first
	if len(p.header) > 0 {
		n := copy(b, p.header)
		p.header = p.header[n:]
		return n, nil
	}

	// then continue reading from the source
	return p.r.Read(b)
}

func (p *HeaderReader) PeekHeader() []byte {
	return p.header
}

// extractor is a private interface and defines all functions that needs to be implemented by an extraction engine.
type extractor interface {
	// Unpack is the main entrypoint to an extraction engine that takes the contents from src and extracts them to dst.
	Unpack(ctx context.Context, src io.Reader, dst string, target target.Target, config *config.Config) error
}

// AvailableExtractors is collection of new extractor functions with
// the required magic bytes and potential offset
var AvailableExtractors = []struct {
	NewExtractor func() extractor
	HeaderCheck  func([]byte) bool
	MagicBytes   [][]byte
	Offset       int
}{
	{
		NewExtractor: func() extractor {
			return NewTar()
		},
		HeaderCheck: IsTar,
		MagicBytes:  magicBytesTar,
		Offset:      offsetTar,
	},
	{
		NewExtractor: func() extractor {
			return NewZip()
		},
		HeaderCheck: IsZip,
		MagicBytes:  magicBytesZIP,
	},
	{
		NewExtractor: func() extractor {
			return NewGzip()
		},
		HeaderCheck: IsGZIP,
		MagicBytes:  magicBytesGZIP,
	},
}

var MaxHeaderLength int

func init() {
	for _, ex := range AvailableExtractors {
		needs := ex.Offset
		for _, mb := range ex.MagicBytes {
			if len(mb)+ex.Offset > needs {
				needs = len(mb) + ex.Offset
			}
		}
		if needs > MaxHeaderLength {
			MaxHeaderLength = needs
		}
	}
}
