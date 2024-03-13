package extractor

import (
	"compress/gzip"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
)

// magicBytesGZip are the magic bytes for gzip compressed files
// reference https://socketloop.com/tutorials/golang-gunzip-file
var magicBytesGZip = [][]byte{
	{0x1f, 0x8b},
}

// fileExtensionGZip is the file extension for gzip files
var fileExtensionGZip = "gz"

// IsGZip checks if the header matches the magic bytes for gzip compressed files
func IsGZip(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZip)
}

// Unpack sets a timeout for the ctx and starts the gzip uncompression from src to dst.
func UnpackGZip(ctx context.Context, src io.Reader, dst string, c *config.Config) error {
	return uncompress(ctx, src, dst, c, uncompressGZipStream, fileExtensionGZip)
}

// uncompressGZipStream returns an io.Reader that uncompresses src with gzip algorithm
func uncompressGZipStream(src io.Reader, c *config.Config) (io.Reader, error) {
	return gzip.NewReader(src)
}
