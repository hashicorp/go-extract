// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"fmt"
	"io"
)

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
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("cannot read header: %w", err)
	}
	return &headerReader{r, buf[:n]}, nil
}

func (p *headerReader) Read(b []byte) (int, error) {
	// read from header first
	if len(p.header) > 0 {
		n := copy(b, p.header)
		p.header = p.header[n:]
		return n, nil
	}

	// then continue reading from the source
	return p.r.Read(b)
}

func (p *headerReader) PeekHeader() []byte {
	return p.header
}
