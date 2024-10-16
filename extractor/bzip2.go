package extractor

import (
	"compress/bzip2"
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// MagicBytesBzip2 are the magic bytes for bzip2 compressed files
// reference: https://en.wikipedia.org/wiki/Bzip2 // https://github.com/dsnet/compress/blob/master/doc/bzip2-format.pdf
var MagicBytesBzip2 = [][]byte{
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

// FileExtensionBzip2 is the file extension for bzip2 files
const FileExtensionBzip2 = "bz2"

// IsBzip2 checks if the header matches the magic bytes for bzip2 compressed files
func IsBzip2(header []byte) bool {
	return matchesMagicBytes(header, 0, MagicBytesBzip2)
}

// Unpack sets a timeout for the ctx and starts the bzip2 decompression from src to dst.
func UnpackBzip2(ctx context.Context, t target.Target, dst string, src io.Reader, cfg *config.Config) error {
	return decompress(ctx, t, dst, src, cfg, decompressBz2Stream, "bz2")
}

func decompressBz2Stream(src io.Reader) (io.Reader, error) {
	return bzip2.NewReader(src), nil
}
