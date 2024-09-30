package extractor

import "io"

// noopReaderCloser is a struct that implements the io.ReaderCloser interface with a no-op Close method.
type noopReaderCloser struct {
	io.Reader
}

// Close is a no-op method that satisfies the io.Closer interface.
func (n *noopReaderCloser) Close() error {
	return nil
}
