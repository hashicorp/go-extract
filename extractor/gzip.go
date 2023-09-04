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
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// reference https://socketloop.com/tutorials/golang-gunzip-file

// Gzip is a struct type that holds all information to perform an gzip decompression
type Gzip struct {

	// config holds ther configuration for the extractor
	config *config.Config

	// fileSuffix holds the common file suffix for this archive type
	fileSuffix string

	// target is the extraction target
	target target.Target

	// magicBytes are the magic bytes that are used to identify a gzip compressed file
	magicBytes [][]byte

	// offset is the offset before the magic bytes can be found
	offset int
}

// NewGzip returns a new Gzip object with config as configuration.
func NewGzip(config *config.Config) *Gzip {
	// defaults
	const (
		fileSuffix = ".gz"
	)
	magicBytes := [][]byte{
		{0x1f, 0x8b},
	}
	offset := 0

	// setup extraction target
	target := target.NewOs()

	// instantiate
	gzip := Gzip{
		fileSuffix: fileSuffix,
		config:     config,
		target:     target,
		magicBytes: magicBytes,
		offset:     offset,
	}

	// return the modified house instance
	return &gzip
}

// FileSuffix returns the common file suffix of gzip archive type.
func (gz *Gzip) FileSuffix() string {
	return gz.fileSuffix
}

// SetConfig sets config as configuration.
func (gz *Gzip) SetConfig(config *config.Config) {
	gz.config = config
}

// SetTarget sets target as a extraction destination
func (gz *Gzip) SetTarget(target target.Target) {
	gz.target = target
}

// Offset returns the offset for the magic bytes.
func (gz *Gzip) Offset() int {
	return gz.offset
}

// MagicBytes returns the magic bytes that identifies gzip files.
func (gz *Gzip) MagicBytes() [][]byte {
	return gz.magicBytes
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func (g *Gzip) Unpack(ctx context.Context, src io.Reader, dst string) error {

	// start extraction without timer
	if g.config.MaxExtractionTime == -1 {
		return g.unpack(ctx, src, dst)
	}

	// prepare timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(g.config.MaxExtractionTime)*time.Second)
	defer cancel()

	exChan := make(chan error, 1)
	go func() {
		// extract files in tmpDir
		if err := g.unpack(ctx, src, dst); err != nil {
			exChan <- err
		}
		exChan <- nil
	}()

	// start extraction in on thread
	select {
	case err := <-exChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("maximum extraction time exceeded")
	}

	return nil
}

// Unpack decompresses src with gzip algorithm into dst. If src is a gziped tar archive,
// the tar archive is extracted
func (gz *Gzip) unpack(ctx context.Context, src io.Reader, dst string) error {

	// open reader
	uncompressedStream, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("cannot decompress gzip")
	}

	// size check
	var bytesBuffer bytes.Buffer
	if gz.config.MaxExtractionSize > -1 {
		var readBytes int64
		for {
			buf := make([]byte, 1024)
			n, err := uncompressedStream.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}

			// cothing read
			if n == 0 {
				break
			}

			// check if maximum is exceeded
			if readBytes+int64(n) < gz.config.MaxExtractionSize {
				bytesBuffer.Write(buf[:n])
				readBytes = readBytes + int64(n)

				// check if context is cancled
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
	tar := NewTar(gz.config)
	if tar.MagicBytesMatch(bytesBuffer.Bytes()) {
		return tar.Unpack(ctx, bytes.NewReader(bytesBuffer.Bytes()), dst)
	}

	// determine name for decompressed content
	name := "gunziped-content"
	if dst != "." {
		if stat, err := os.Stat(dst); os.IsNotExist(err) || stat.Mode()&fs.ModeDir != 0 {
			name = filepath.Base(dst)
			dst = filepath.Dir(dst)
		}
	}

	// check if context is cancled
	if ctx.Err() != nil {
		return nil
	}

	// Create file
	return gz.target.CreateSafeFile(gz.config, dst, name, bytes.NewReader(bytesBuffer.Bytes()), 0644)
}
