package extractor

import (
	"compress/bzip2"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
)

// magicBytesBzip2 are the magic bytes for bzip2 compressed files
// reference: https://en.wikipedia.org/wiki/Bzip2 // https://github.com/dsnet/compress/blob/master/doc/bzip2-format.pdf
var magicBytesBzip2 = [][]byte{
	[]byte("BZh1"),
	[]byte("BZh2"),
	[]byte("BZh3"),
	[]byte("BZh4"),
	[]byte("BZh5"),
	[]byte("BZh6"),
	[]byte("BZh7"),
	[]byte("BZh8"),
	[]byte("BZh9"),
}

// fileExtensionBzip2 is the file extension for bzip2 files
var fileExtensionBzip2 = "bz2"

// IsBzip2 checks if the header matches the magic bytes for bzip2 compressed files
func IsBzip2(header []byte) bool {
	return matchesMagicBytes(header, 0, magicBytesBzip2)
}

// Unpack sets a timeout for the ctx and starts the bzip2 uncompression from src to dst.
func UnpackBzip2(ctx context.Context, src io.Reader, dst string, c *config.Config) error {
	return uncompress(ctx, src, dst, c, uncompressBz2Stream, "bz2")
}

func uncompressBz2Stream(src io.Reader, c *config.Config) (io.Reader, error) {
	return bzip2.NewReader(src), nil
}
