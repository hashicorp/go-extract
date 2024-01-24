package extract

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
	"github.com/hashicorp/go-extract/target"
)

// extractorsForMagicBytes is collection of new extractor functions with
// the required magic bytes and potential offset
var extractorsForMagicBytes = []struct {
	newExtractor func() Extractor
	offset       int
	magicBytes   [][]byte
}{
	{
		newExtractor: func() Extractor {
			return extractor.NewTar()
		},
		offset:     extractor.OffsetTar,
		magicBytes: extractor.MagicBytesTar,
	},
	{
		newExtractor: func() Extractor {
			return extractor.NewZip()
		},
		magicBytes: extractor.MagicBytesZIP,
	},
	{
		newExtractor: func() Extractor {
			return extractor.NewGzip()
		},
		magicBytes: extractor.MagicBytesGZIP,
	},
}

var headerLength int

func init() {
	for _, ex := range extractorsForMagicBytes {
		needs := ex.offset
		for _, mb := range ex.magicBytes {
			needs += len(mb)
		}
		if needs > headerLength {
			headerLength = needs
		}
	}
}

// headerReader is an implementation of io.Reader that allows the first bytes of
// the reader to be read twice. This is useful for identifying the archive type
// before unpacking.
type headerReader struct {
	r      io.Reader
	header []byte
}

func newHeaderReader(r io.Reader, headerSize int) (*headerReader, error) {
	// read at least headerSize bytes. If EOF, capture whatever was read.
	buf := make([]byte, headerSize)
	n, err := io.ReadAtLeast(r, buf, headerSize)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return &headerReader{r, buf[:n]}, nil
}

func (p *headerReader) Read(b []byte) (n int, err error) {
	// read from header first
	if len(p.header) > 0 {
		n = copy(b, p.header)
		p.header = p.header[n:]
		return n, nil
	}

	// then continue reading from the source
	return p.r.Read(b)
}

func (p *headerReader) PeekHeader() []byte {
	return p.header
}

// Unpack reads data from src, identifies if its a known archive type. If so, dst is unpacked
// in dst. opts can be given to adjust the config and target.
func Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
	var ex Extractor

	header, err := newHeaderReader(src, headerLength)
	if err != nil {
		return err
	}
	headerData := header.PeekHeader()

	if ex = findExtractor(headerData); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// limit input file size if configured
	src = NewLimitErrorReader(src, c.MaxInputFileSize)

	// perform extraction with identified reader
	return ex.Unpack(ctx, header, dst, t, c)
}

// findExtractor identifies the correct extractor based on magic bytes.
func findExtractor(data []byte) Extractor {
	// find extractor with longest suffix match
	for _, ex := range extractorsForMagicBytes {
		// check all possible magic bytes for extract engine
		for _, magicBytes := range ex.magicBytes {

			// check for byte match
			if matchesMagicBytes(data, ex.offset, magicBytes) {
				return ex.newExtractor()
			}
		}
	}

	// no matching reader found
	return nil
}

// matchesMagicBytes checks if the bytes in data are equal to magicBytes after at a given offset
func matchesMagicBytes(data []byte, offset int, magicBytes []byte) bool {
	if offset+len(magicBytes) > len(data) {
		return false
	}

	return bytes.Equal(magicBytes, data[offset:offset+len(magicBytes)])
}

// LimitErrorReader is a reader that returns an error if the limit is exceeded
// before the underlying reader is fully read.
// If the limit is -1, all data from the original reader is read.
type LimitErrorReader struct {
	O io.Reader // original reader
	R io.Reader // limited underlying reader
}

// Read reads from the underlying reader and returns an error if the limit is exceeded
// before the underlying reader is fully read.
// If the limit is -1, all data from the original reader is read.
// If the limit is exceeded, the original reader is read to check if more data is available.
// If more data is available, an error is returned and left over data is stored in the buffer.
func (l *LimitErrorReader) Read(p []byte) (int, error) {
	n, err := l.R.Read(p)

	// check if original source is also fully read
	if n == 0 {
		if n, err = l.O.Read(p); n > 0 {
			return 0, fmt.Errorf("read limit exceeded, but more data available")
		}
	}

	// return
	return n, err
}

// NewLimitErrorReader returns a new LimitErrorReader that reads from r
func NewLimitErrorReader(r io.Reader, limit int64) *LimitErrorReader {
	if limit > -1 {
		return &LimitErrorReader{
			O: r,
			R: io.LimitReader(r, limit),
		}
	}
	return &LimitErrorReader{
		O: r,
		R: r,
	}
}
