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

// gzipReader is a global variable to avoid creating a new reader for each file
// WARNING: this is not thread safe
var gzipReader *gzip.Reader

// IsGZip checks if the header matches the magic bytes for gzip compressed files
func IsGZip(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesGZip)
}

// Unpack sets a timeout for the ctx and starts the gzip decompression from src to dst.
func UnpackGZip(ctx context.Context, src io.Reader, dst string, c *config.Config) error {
	return decompress(ctx, src, dst, c, decompressGZipStream, fileExtensionGZip)
}

// decompressGZipStream returns an io.Reader that decompresses src with gzip algorithm
func decompressGZipStream(src io.Reader, c *config.Config) (io.Reader, error) {

	var err error
	// reset the reader to use the new source
	if gzipReader != nil {
		err = gzipReader.Reset(src)
	}

	// create a new gzip reader, or reuse the existing one
	if gzipReader == nil {
		gzipReader, err = gzip.NewReader(src)
	}

	return gzipReader, err
}
