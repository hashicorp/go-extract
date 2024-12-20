// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"io"

	"github.com/andybalholm/brotli"
)

// fileExtensionBrotli is the file extension for brotli files
const fileExtensionBrotli = "br"

// isBrotli returns always false, because the brotli magic bytes are not unique
func isBrotli(header []byte) bool {
	return false
}

// Unpack sets a timeout for the ctx and starts the brotli decompression from src to dst.
func unpackBrotli(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressBrotliStream, "br")
}

// decompressBrotliStream returns an io.Reader that decompresses src with brotli algorithm
func decompressBrotliStream(src io.Reader) (io.Reader, error) {
	return brotli.NewReader(src), nil
}
