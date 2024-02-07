package config

import (
	"fmt"
	"io"
)

// LimitErrorReader is a reader that returns an error if the limit is exceeded
// before the underlying reader is fully read.
// If the limit is -1, all data from the original reader is read.
type LimitErrorReader struct {
	R io.Reader // underlying reader
	L int64     // limit
	N int64     // number of bytes read
}

// Read reads from the underlying reader and fills up p.
// It returns an error if the limit is exceeded, even if the underlying reader is not fully read.
// If the limit is -1, all data from the original reader is read.
// Remark: Even if the limit is exceeded, the buffer p is filled up to the max or until the underlying
// reader is fully read.
func (l *LimitErrorReader) Read(p []byte) (int, error) {

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
func (l *LimitErrorReader) ReadBytes() int {
	return int(l.N)
}

// NewLimitErrorReader returns a new LimitErrorReader that reads from r
func NewLimitErrorReader(r io.Reader, limit int64) *LimitErrorReader {
	return &LimitErrorReader{R: r, L: limit, N: 0}
}
