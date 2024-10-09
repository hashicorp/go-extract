package extractor

import (
	"context"
	"io"

	"github.com/golang/snappy"
	"github.com/hashicorp/go-extract/config"
)

// magicBytesSnappy is the magic bytes for snappy files.
var magicBytesSnappy = [][]byte{
	append([]byte{0xff, 0x06, 0x00, 0x00}, []byte("sNaPpY")...),
}

// FileExtensionSnappy is the file extension for snappy files.
const FileExtensionSnappy = "sz"

// isSnappy checks if the header matches the snappy magic bytes.
func isSnappy(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesSnappy)
}

// Unpack sets a timeout for the ctx and starts the snappy decompression from src to dst.
func UnpackSnappy(ctx context.Context, t Target, dst string, src io.Reader, cfg *config.Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressSnappyStream, FileExtensionSnappy)
}

// decompressSnappyStream returns an io.Reader that decompresses src with snappy algorithm
func decompressSnappyStream(src io.Reader) (io.Reader, error) {
	return snappy.NewReader(src), nil
}
