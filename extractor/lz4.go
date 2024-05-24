package extractor

import (
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
	"github.com/pierrec/lz4/v4"
)

// magicBytesLZ4 is the magic bytes for LZ4 files.
// reference https://android.googlesource.com/platform/external/lz4/+/HEAD/doc/lz4_Frame_format.md
var magicBytesLZ4 = [][]byte{
	{0x04, 0x22, 0x4D, 0x18},
}

// FileExtensionLZ4 is the file extension for LZ4 files.
const FileExtensionLZ4 = "lz4"

// IsLZ4 checks if the header matches the LZ4 magic bytes.
func IsLZ4(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesLZ4)
}

// Unpack sets a timeout for the ctx and starts the lz4 decompression from src to dst.
func UnpackLZ4(ctx context.Context, t target.Target, dst string, src io.Reader, c *config.Config) error {
	return decompress(ctx, t, dst, src, c, decompressLZ4Stream, FileExtensionLZ4)
}

// decompressZlibStream returns an io.Reader that decompresses src with zlib algorithm
func decompressLZ4Stream(src io.Reader, c *config.Config) (io.Reader, error) {
	return lz4.NewReader(src), nil
}
