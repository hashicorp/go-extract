package extractor

import (
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/ulikunitz/xz"
)

// magicBytesXz is the magic bytes for xz files.
// reference https://tukaani.org/xz/xz-file-format-1.0.4.txt
var magicBytesXz = [][]byte{
	{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
}

// FileExtensionXz is the file extension for xz files.
const FileExtensionXz = "xz"

// IsXz checks if the header matches the xz magic bytes.
func IsXz(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesXz)
}

// Unpack sets a timeout for the ctx and starts the xz decompression from src to dst.
func UnpackXz(ctx context.Context, src io.Reader, dst string, c *config.Config) error {
	return decompress(ctx, src, dst, c, decompressXzStream, FileExtensionXz)
}

// decompressZlibStream returns an io.Reader that decompresses src with xz algorithm
func decompressXzStream(src io.Reader, c *config.Config) (io.Reader, error) {
	return xz.NewReader(src)
}
