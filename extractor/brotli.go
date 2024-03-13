package extractor

import (
	"context"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/hashicorp/go-extract/config"
)

// magicBytesBrotli are the magic bytes for brotli compressed files
// reference https://github.com/madler/brotli/blob/1d428d3a9baade233ebc3ac108293256bcb813d1/br-format-v3.txt#L114-L116
var magicBytesBrotli = [][]byte{
	{0xce, 0xb2, 0xcf, 0x81},
}

// fileExtensionBrotli is the file extension for brotli files
var fileExtensionBrotli = "br"

// IsBrotli checks if the header matches the magic bytes for brotli compressed files
func IsBrotli(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesBrotli)
}

// Unpack sets a timeout for the ctx and starts the brotli uncompression from src to dst.
func UnpackBrotli(ctx context.Context, src io.Reader, dst string, c *config.Config) error {
	return uncompress(ctx, src, dst, c, uncompressBrotliStream, "br")
}

// uncompressBrotliStream returns an io.Reader that uncompresses src with brotli algorithm
func uncompressBrotliStream(src io.Reader, c *config.Config) (io.Reader, error) {
	return brotli.NewReader(src), nil
}
