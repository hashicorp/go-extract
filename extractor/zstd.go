package extractor

import (
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/klauspost/compress/zstd"
)

// magicBytesZstd is the magic bytes for zstandard files.
// reference: https://www.rfc-editor.org/rfc/rfc8878.html
var magicBytesZstd = [][]byte{
	{0x28, 0xb5, 0x2f, 0xfd},
}

// fileExtensionZstd is the file extension for zstandard files.
var fileExtensionZstd = "zst"

// IsZstd checks if the header matches the zstandard magic bytes.
func IsZstd(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesZstd)
}

// Unpack sets a timeout for the ctx and starts the zstandard decompression from src to dst.
func UnpackZstd(ctx context.Context, src io.Reader, dst string, c *config.Config) error {
	return decompress(ctx, src, dst, c, decompressZstdStream, fileExtensionZstd)
}

// decompressZstdStream returns an io.Reader that decompresses src with zstandard algorithm
func decompressZstdStream(src io.Reader, c *config.Config) (io.Reader, error) {
	return zstd.NewReader(src)
}