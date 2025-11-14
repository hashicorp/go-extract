// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"io"

	"github.com/klauspost/compress/zstd"
)

// fileExtensionZstd is the file extension for zstandard files.
const fileExtensionZstd = "zst"

// magicBytesZstd is the magic bytes for zstandard files.
// reference: https://www.rfc-editor.org/rfc/rfc8878.html
var magicBytesZstd = [][]byte{
	{0x28, 0xb5, 0x2f, 0xfd},
}

// isZstd checks if the header matches the zstandard magic bytes.
func isZstd(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZstd)
}

// Unpack sets a timeout for the ctx and starts the zstandard decompression from src to dst.
func unpackZstd(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressZstdStream, fileExtensionZstd)
}

// decompressZstdStream returns an io.Reader that decompresses src with zstandard algorithm
func decompressZstdStream(src io.Reader) (io.Reader, error) {
	return zstd.NewReader(src)
}
