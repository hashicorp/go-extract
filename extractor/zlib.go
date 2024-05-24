package extractor

import (
	"compress/zlib"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
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

// FileExtensionZlib is the file extension for Zlib files.
const FileExtensionZlib = "zz"

// IsZlib checks if the header matches the Zlib magic bytes.
func IsZlib(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZlib)
}

// Unpack sets a timeout for the ctx and starts the zlib decompression from src to dst.
func UnpackZlib(ctx context.Context, t target.Target, dst string, src io.Reader, c *config.Config) error {
	return decompress(ctx, t, dst, src, c, decompressZlibStream, FileExtensionZlib)
}

// decompressZlibStream returns an io.Reader that decompresses src with zlib algorithm
func decompressZlibStream(src io.Reader, c *config.Config) (io.Reader, error) {
	return zlib.NewReader(src)
}
