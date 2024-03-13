package extractor

import (
	"compress/zlib"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
)

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

// fileExtensionZlib is the file extension for Zlib files.
var fileExtensionZlib = "zz"

// IsZlib checks if the header matches the Zlib magic bytes.
func IsZlib(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZlib)
}

// Unpack sets a timeout for the ctx and starts the zlib uncompression from src to dst.
func UnpackZlib(ctx context.Context, src io.Reader, dst string, c *config.Config) error {
	return uncompress(ctx, src, dst, c, uncompressZlibStream, fileExtensionZlib)
}

// uncompressZlibStream returns an io.Reader that uncompresses src with zlib algorithm
func uncompressZlibStream(src io.Reader, c *config.Config) (io.Reader, error) {
	return zlib.NewReader(src)
}
