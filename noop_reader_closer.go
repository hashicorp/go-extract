// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package extract

import "io"

// noopReaderCloser is a struct that implements the io.ReaderCloser interface with a no-op Close method.
type noopReaderCloser struct {
	io.Reader
}

// Close is a no-op method that satisfies the io.Closer interface.
func (n *noopReaderCloser) Close() error {
	return nil
}
