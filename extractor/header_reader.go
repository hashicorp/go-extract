package extractor

import (
	"fmt"
	"io"
)

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
		return nil, fmt.Errorf("cannot read header: %s", err)
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
