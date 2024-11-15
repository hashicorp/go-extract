// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"compress/gzip"
	"context"
	"io"
)

const (
	// FileExtensionGZip is the file extension for gzip files.
	FileExtensionGZip = "gz"

	// FileExtensionTarGZip is the file extension for tgz files, which are tar archives compressed with gzip.
	FileExtensionTarGZip = "tgz"
)

// magicBytesGZip are the magic bytes for gzip compressed files.
//
// https://socketloop.com/tutorials/golang-gunzip-file
var magicBytesGZip = [][]byte{
	{0x1f, 0x8b},
}

// IsGZip checks if the header matches the magic bytes for gzip compressed files.
func IsGZip(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZip)
}

// Unpack sets a timeout for the ctx and starts the gzip decompression from src to dst.
func UnpackGZip(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressGZipStream, FileExtensionGZip)
}

// decompressGZipStream returns an io.Reader that decompresses src with gzip algorithm.
func decompressGZipStream(src io.Reader) (io.Reader, error) {
	return gzip.NewReader(src)
}
