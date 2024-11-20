// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"compress/zlib"
	"context"
	"io"
)

// fileExtensionZlib is the file extension for Zlib files.
const fileExtensionZlib = "zz"

// magicBytesZlib is the magic bytes for Zlib files.
// reference https://www.ietf.org/rfc/rfc1950.txt
var magicBytesZlib = [][]byte{
	{0x78, 0x01},
	{0x78, 0x5e},
	{0x78, 0x9c},
	{0x78, 0xda},
	{0x78, 0x20},
	{0x78, 0x7d},
	{0x78, 0xbb},
	{0x78, 0xf9},
}

// isZlib checks if the header matches the Zlib magic bytes.
func isZlib(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZlib)
}

// Unpack sets a timeout for the ctx and starts the zlib decompression from src to dst.
func unpackZlib(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressZlibStream, fileExtensionZlib)
}

// decompressZlibStream returns an io.Reader that decompresses src with zlib algorithm
func decompressZlibStream(src io.Reader) (io.Reader, error) {
	return zlib.NewReader(src)
}
