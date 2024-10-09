package extractor

import (
	"compress/zlib"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
)

// FileExtensionZlib is the file extension for Zlib files.
const FileExtensionZlib = "zz"

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
func UnpackZlib(ctx context.Context, t Target, dst string, src io.Reader, cfg *config.Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressZlibStream, FileExtensionZlib)
}

// decompressZlibStream returns an io.Reader that decompresses src with zlib algorithm
func decompressZlibStream(src io.Reader) (io.Reader, error) {
	return zlib.NewReader(src)
}