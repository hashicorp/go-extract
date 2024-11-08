package extract

import (
	"context"
	"io"

	"github.com/ulikunitz/xz"
)

// FileExtensionXz is the file extension for xz files.
const FileExtensionXz = "xz"

// magicBytesXz is the magic bytes for xz files.
// reference https://tukaani.org/xz/xz-file-format-1.0.4.txt
var magicBytesXz = [][]byte{
	{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
}

// IsXz checks if the header matches the xz magic bytes.
func IsXz(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesXz)
}

// Unpack sets a timeout for the ctx and starts the xz decompression from src to dst.
func UnpackXz(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressXzStream, FileExtensionXz)
}

// decompressZlibStream returns an io.Reader that decompresses src with xz algorithm
func decompressXzStream(src io.Reader) (io.Reader, error) {
	return xz.NewReader(src)
}
