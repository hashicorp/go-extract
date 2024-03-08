package target

import "io"

// LimitErrorWrite is a wrapper around an io.Writer that returns io.ErrShortWrite
// when the limit is reached.
type LimitErrorWrite struct {
	W io.Writer // underlying reader
	L int64     // limit
	N int64     // number of bytes read
}

// Write writes up to len(p) bytes from p to the underlying data stream. It returns
// the number of bytes written from p (0 <= n <= len(p)) and any error encountered
// that caused the write to stop early. Write returns a non-nil error when n < len(p).
// Write does not modify the slice data, even temporarily. The limit is enforced by
// returning io.ErrShortWrite when the limit is reached.
func (l *LimitErrorWrite) Write(p []byte) (n int, err error) {

	// check if we reached the limit
	if l.N >= l.L {
		return 0, io.ErrShortWrite
	}

	// write until we reach the limit
	if int64(len(p)) > l.L-l.N {
		p = p[0 : l.L-l.N]
		n, err = l.W.Write(p)
		if err == nil {
			err = io.ErrShortWrite
		}
		l.N += int64(n)
		return n, err
	}

	// write normally
	n, err = l.W.Write(p)
	l.N += int64(n)
	return n, err
}

// NewLimitErrorWrite returns a new LimitErrorWrite that wraps the given writer
// and limit.
func NewLimitErrorWrite(w io.Writer, l int64) *LimitErrorWrite {
	return &LimitErrorWrite{W: w, L: l}
}
