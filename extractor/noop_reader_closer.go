package extractor

import "io"

// NoopReaderCloser is a struct that implements the io.ReaderCloser interface with a no-op Close method.
type NoopReaderCloser struct {
	io.Reader
}

// Close is a no-op method that satisfies the io.Closer interface.
func (n *NoopReaderCloser) Close() error {
	return nil
}
