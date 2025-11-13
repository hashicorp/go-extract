// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"fmt"
	"io"
)

// limitErrorReader is a reader that returns an error if the limit is exceeded
// before the underlying reader is fully read.
// If the limit is -1, all data from the original reader is read.
type limitErrorReader struct {
	R io.Reader // underlying reader
	L int64     // limit
	N int64     // number of bytes read
}

// Read reads from the underlying reader and fills up p.
// It returns an error if the limit is exceeded, even if the underlying reader is not fully read.
// If the limit is -1, all data from the original reader is read.
func (l *limitErrorReader) Read(p []byte) (int, error) {
	// determine how many bytes to read
	m := l.L - l.N
	if l.L == -1 || m > int64(len(p)) {
		m = int64(len(p))
	}

	// check if limit has exceeded
	if m == 0 {
		return 0, fmt.Errorf("read limit exceeded")
	}

	// read from underlying reader and preserve error type
	n, err := l.R.Read(p[:m])
	l.N += int64(n)
	return n, err
}

// ReadBytes returns how many bytes have been read from the underlying reader
func (l *limitErrorReader) ReadBytes() int {
	return int(l.N)
}

// newLimitErrorReader returns a new LimitErrorReader that reads from r
func newLimitErrorReader(r io.Reader, limit int64) *limitErrorReader {
	return &limitErrorReader{R: r, L: limit, N: 0}
}
