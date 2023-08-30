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

// Gunzip is a struct type that holds all information to perform an gunzip decompression
type Gunzip struct {

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

// NewZip returns a new Gunzip object with config as configuration.
func NewGunzip(config *config.Config) *Gunzip {
	// defaults
	const (
		fileSuffix = ".gz"
	)
	magicBytes := [][]byte{
		{0x1f, 0x8b},
	}
	offset := 0

	// setup extraction target
	target := target.NewOs(config)

	// instantiate
	gunzip := Gunzip{
		fileSuffix: fileSuffix,
		config:     config,
		target:     target,
		magicBytes: magicBytes,
		offset:     offset,
	}

	// return the modified house instance
	return &gunzip
}

// FileSuffix returns the common file suffix of gzip archive type.
func (gz *Gunzip) FileSuffix() string {
	return gz.fileSuffix
}

// SetConfig sets config as configuration.
func (gz *Gunzip) SetConfig(config *config.Config) {
	gz.config = config
	gz.target.SetConfig(config)
}

// SetTarget sets target as a extraction destination
func (gz *Gunzip) SetTarget(target *target.Target) {
	gz.target = *target
}

// Offset returns the offset for the magic bytes.
func (gz *Gunzip) Offset() int {
	return gz.offset
}

// MagicBytes returns the magic bytes that identifies gzip files.
func (gz *Gunzip) MagicBytes() [][]byte {
	return gz.magicBytes
}

// Unpack decompresses src with gzip algorithm into dst. If src is a gziped tar archive,
// the tar archive is extracted
func (gz *Gunzip) Unpack(ctx context.Context, src io.Reader, dst string) error {

	// open reader
	uncompressedStream, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("cannot decompress gunzip")
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

			}
		}

	} else {
		_, err = bytesBuffer.ReadFrom(uncompressedStream)
		if err != nil {
			return fmt.Errorf("cannot read decompressed gunzip")
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
	return gz.target.CreateSafeFile(dst, name, bytes.NewReader(bytesBuffer.Bytes()), 0644)
}
