package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// extractor is a private interface and defines all functions that needs to be implemented by an extraction engine.
type extractor interface {
	// Unpack is the main entrypoint to an extraction engine that takes the contents from src and extracts them to dst.
	Unpack(ctx context.Context, src io.Reader, dst string, target target.Target, config *config.Config) error
}

// prepare ensures limited read and generic metric capturing
// remark: this preparation is located in the extractor package so that the
// different extractor engines can be used independently and keep their
// functionality.
func prepare(ctx context.Context, src io.Reader, c *config.Config) io.Reader {

	// setup reader and timer
	start := time.Now()                                      // capture start to calculate execution time
	ler := newLimitErrorReaderCounter(src, c.MaxInputSize()) // ensure input size and capture metrics

	// extend metric collection
	c.AddMetricsHook(func(ctx context.Context, m *config.Metrics) {
		m.ExtractionDuration = time.Since(start) // capture execution time
		m.InputSize = int64(ler.ReadBytes())     // capture inputSize metric
	})

	return ler
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

// limitErrorReaderCounter is a reader that returns an error if the limit is exceeded
// before the underlying reader is fully read.
// If the limit is -1, all data from the original reader is read.
type limitErrorReaderCounter struct {
	R io.Reader // underlying reader
	L int64     // limit
	N int64     // number of bytes read
}

// Read reads from the underlying reader and fills up p.
// It returns an error if the limit is exceeded, even if the underlying reader is not fully read.
// If the limit is -1, all data from the original reader is read.
// Remark: Even if the limit is exceeded, the buffer p is filled up to the max or until the underlying
// reader is fully read.
func (l *limitErrorReaderCounter) Read(p []byte) (int, error) {

	// read from underlying reader
	n, err := l.R.Read(p)
	l.N += int64(n)
	if err != nil {
		return n, err
	}

	// check if limit has exceeded
	if l.L >= 0 && l.N > l.L {
		return n, fmt.Errorf("read limit exceeded")
	}

	// return
	return n, nil
}

// ReadBytes returns how many bytes have been read from the underlying reader
func (l *limitErrorReaderCounter) ReadBytes() int {
	return int(l.N)
}

// newLimitErrorReaderCounter returns a new limitErrorReaderCounter that reads from r
func newLimitErrorReaderCounter(r io.Reader, limit int64) *limitErrorReaderCounter {
	return &limitErrorReaderCounter{R: r, L: limit, N: 0}
}

// handleError increases the error counter, sets the latest error and
// decides if extraction should continue.
func handleError(c *config.Config, metrics *config.Metrics, msg string, err error) error {

	// increase error counter and set error
	metrics.ExtractionErrors++
	metrics.LastExtractionError = fmt.Errorf("%s: %s", msg, err)

	// do not end on error
	if c.ContinueOnError() {
		c.Logger().Error(msg, "error", err)
		return nil
	}

	// end extraction on error
	return metrics.LastExtractionError
}
