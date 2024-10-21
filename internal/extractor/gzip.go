package extractor

import (
	"compress/gzip"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
)

const (
	// FileExtensionGZip is the file extension for gzip files.
	FileExtensionGZip = "gz"

	// FileExtensionTarGZip is the file extension for tgz files, which are tar archives compressed with gzip.
	FileExtensionTarGZip = "tgz"
)

// magicBytesGZip are the magic bytes for gzip compressed files.
//
// https://socketloop.com/tutorials/golang-gunzip-file
var magicBytesGZip = [][]byte{
	{0x1f, 0x8b},
}

// isGZip checks if the header matches the magic bytes for gzip compressed files.
func isGZip(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZip)
}

// Unpack sets a timeout for the ctx and starts the gzip decompression from src to dst.
func UnpackGZip(ctx context.Context, t Target, dst string, src io.Reader, cfg *config.Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressGZipStream, FileExtensionGZip)
}

// decompressGZipStream returns an io.Reader that decompresses src with gzip algorithm.
func decompressGZipStream(src io.Reader) (io.Reader, error) {
	return gzip.NewReader(src)
}
