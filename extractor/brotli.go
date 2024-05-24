package extractor

import (
	"context"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// FileExtensionBrotli is the file extension for brotli files
const FileExtensionBrotli = "br"

// IsBrotli returns always false, because the brotli magic bytes are not unique
func IsBrotli(header []byte) bool {
	return false
}

// Unpack sets a timeout for the ctx and starts the brotli decompression from src to dst.
func UnpackBrotli(ctx context.Context, t target.Target, dst string, src io.Reader, c *config.Config) error {
	return decompress(ctx, t, dst, src, c, decompressBrotliStream, "br")
}

// decompressBrotliStream returns an io.Reader that decompresses src with brotli algorithm
func decompressBrotliStream(src io.Reader, c *config.Config) (io.Reader, error) {
	return brotli.NewReader(src), nil
}
