// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"io"

	"github.com/klauspost/compress/snappy"
)

// fileExtensionSnappy is the file extension for snappy files.
const fileExtensionSnappy = "sz"

// magicBytesSnappy is the magic bytes for snappy files.
var magicBytesSnappy = [][]byte{
	append([]byte{0xff, 0x06, 0x00, 0x00}, []byte("sNaPpY")...),
}

// isSnappy checks if the header matches the snappy magic bytes.
func isSnappy(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesSnappy)
}

// Unpack sets a timeout for the ctx and starts the snappy decompression from src to dst.
func unpackSnappy(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressSnappyStream, fileExtensionSnappy)
}

// decompressSnappyStream returns an io.Reader that decompresses src with snappy algorithm
func decompressSnappyStream(src io.Reader) (io.Reader, error) {
	return snappy.NewReader(src), nil
}
