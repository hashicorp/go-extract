package extractor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

type Gunzip struct {
	config     *config.Config
	fileSuffix string
	target     target.Target
	magicBytes [][]byte
	offset     int
}

func NewGunzip(config *config.Config) *Gunzip {
	// defaults
	const (
		fileSuffix = ".gz"
	)
	magicBytes := [][]byte{
		{0x1f, 0x8b},
	}
	offset := 0

	target := target.NewOs()

	// instantiate
	gunzip := Gunzip{
		fileSuffix: fileSuffix,
		config:     config,
		target:     &target,
		magicBytes: magicBytes,
		offset:     offset,
	}

	// return the modified house instance
	return &gunzip
}

func (gz *Gunzip) FileSuffix() string {
	return gz.fileSuffix
}

func (gz *Gunzip) SetConfig(config *config.Config) {
	gz.config = config
}

func (gz *Gunzip) SetTarget(target *target.Target) {
	gz.target = *target
}

func (gz *Gunzip) Offset() int {
	return gz.offset
}

func (gz *Gunzip) MagicBytes() [][]byte {
	return gz.magicBytes
}

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

	// Create file
	return gz.target.CreateSafeFile(gz.config, dst, uncompressedStream.Header.Name, bytes.NewReader(bytesBuffer.Bytes()), 0)

}
