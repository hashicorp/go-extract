package extractor

import (
	"compress/gzip"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// magicBytesGZip are the magic bytes for gzip compressed files
// reference https://socketloop.com/tutorials/golang-gunzip-file
var magicBytesGZip = [][]byte{
	{0x1f, 0x8b},
}

// FileExtensionGZip is the file extension for gzip files
const FileExtensionGZip = "gz"

// FileExtensionTarGZip is the file extension for tgz files, which are tar archives compressed with gzip
const FileExtensionTarGZip = "tgz"

// IsGZip checks if the header matches the magic bytes for gzip compressed files
func IsGZip(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZip)
}

// Unpack sets a timeout for the ctx and starts the gzip decompression from src to dst.
func UnpackGZip(ctx context.Context, t target.Target, dst string, src io.Reader, cfg *config.Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressGZipStream, FileExtensionGZip)
}

// decompressGZipStream returns an io.Reader that decompresses src with gzip algorithm
func decompressGZipStream(src io.Reader) (io.Reader, error) {
	return gzip.NewReader(src)
}
