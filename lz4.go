// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"io"

	lz4 "github.com/pierrec/lz4/v4"
)

// fileExtensionLZ4 is the file extension for LZ4 files.
const fileExtensionLZ4 = "lz4"

// magicBytesLZ4 is the magic bytes for LZ4 files.
// reference https://android.googlesource.com/platform/external/lz4/+/HEAD/doc/lz4_Frame_format.md
var magicBytesLZ4 = [][]byte{
	{0x04, 0x22, 0x4D, 0x18},
}

// isLZ4 checks if the header matches the LZ4 magic bytes.
func isLZ4(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesLZ4)
}

// Unpack sets a timeout for the ctx and starts the lz4 decompression from src to dst.
func unpackLZ4(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressLZ4Stream, fileExtensionLZ4)
}

// decompressZlibStream returns an io.Reader that decompresses src with zlib algorithm
func decompressLZ4Stream(src io.Reader) (io.Reader, error) {
	return lz4.NewReader(src), nil
}
