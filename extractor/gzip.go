package extractor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// reference https://socketloop.com/tutorials/golang-gunzip-file

var MagicBytesGZIP = [][]byte{
	{0x1f, 0x8b},
}

// Gzip is a struct type that holds all information to perform an gzip decompression
type Gzip struct{}

// NewGzip returns a new Gzip object with config as configuration.
func NewGzip() *Gzip {
	// instantiate
	gzip := Gzip{}

	// return the modified house instance
	return &gzip
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func (g *Gzip) Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
	return g.unpack(ctx, src, dst, t, c)
}

// Unpack decompresses src with gzip algorithm into dst. If src is a gziped tar archive,
// the tar archive is extracted
func (gz *Gzip) unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
	uncompressedStream, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("cannot decompress gzip")
	}

	// size check
	var bytesBuffer bytes.Buffer
	if c.MaxExtractionSize > -1 {
		var readBytes int64
		for {
			buf := make([]byte, 1024)
			n, err := uncompressedStream.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}

			// clothing read
			if n == 0 {
				break
			}

			// check if maximum is exceeded
			if readBytes+int64(n) < c.MaxExtractionSize {
				bytesBuffer.Write(buf[:n])
				readBytes = readBytes + int64(n)

				// check if context is canceled
				if ctx.Err() != nil {
					return nil
				}
			} else {
				return fmt.Errorf("maximum extraction size exceeded")
			}
		}
	} else {
		_, err = bytesBuffer.ReadFrom(uncompressedStream)
		if err != nil {
			return fmt.Errorf("cannot read decompressed gzip")
		}
	}

	// check if src is a tar archive

	for _, magicBytes := range MagicBytesTar {
		if bytes.Equal(magicBytes, bytesBuffer.Bytes()) {
			tar := NewTar()
			return tar.Unpack(ctx, bytes.NewReader(bytesBuffer.Bytes()), dst, t, c)
		}
	}

	// determine name for decompressed content
	name := "gunziped-content"
	if dst != "." {
		if stat, err := os.Stat(dst); os.IsNotExist(err) || stat.Mode()&fs.ModeDir == 0 {
			name = filepath.Base(dst)
			dst = filepath.Dir(dst)
		}
	}

	// check if context is canceled
	if ctx.Err() != nil {
		return nil
	}

	// Create file
	return t.CreateSafeFile(c, dst, name, bytes.NewReader(bytesBuffer.Bytes()), 0644)
}
