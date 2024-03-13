package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

// compressFunc is a function that compresses a byte slice
type compressFunc func([]byte) []byte

func TestUncompress(t *testing.T) {

	ctx := context.Background()
	cfg := config.NewConfig()
	fileContent := []byte("Hello, World!")
	filename := "test"
	testTar := packTarWithContent([]tarContent{{Content: fileContent, Name: filename, Mode: 0640, Filetype: tar.TypeReg}})

	tests := []struct {
		name       string
		comp       compressFunc
		uncompress uncompressionFunction
		ext        string
	}{
		{
			name:       "gzip",
			comp:       compressGzip,
			uncompress: uncompressGZipStream,
			ext:        fileExtensionGZip,
		},
		{
			name:       "zstd",
			comp:       compressZstd,
			uncompress: uncompressZstdStream,
			ext:        fileExtensionZstd,
		},
		{
			name:       "bzip2",
			comp:       compressBzip2,
			uncompress: uncompressBz2Stream,
			ext:        fileExtensionBzip2,
		},
		{
			name:       "xz",
			comp:       compressXz,
			uncompress: uncompressXzStream,
			ext:        fileExtensionXz,
		},
		{
			name:       "brotli",
			comp:       compressBrotli,
			uncompress: uncompressBrotliStream,
			ext:        fileExtensionBrotli,
		},
		{
			name:       "lz4",
			comp:       compressLZ4,
			uncompress: uncompressLZ4Stream,
			ext:        fileExtensionLZ4,
		},
		{
			name:       "snappy",
			comp:       compressSnappy,
			uncompress: uncompressSnappyStream,
			ext:        fileExtensionSnappy,
		},
		{
			name:       "zlib",
			comp:       compressZlib,
			uncompress: uncompressZlibStream,
			ext:        fileExtensionZlib,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test.tar.%s", tt.ext))
			r := createFile(testFile, tt.comp(testTar))
			defer func() {
				if f, ok := r.(io.Closer); ok {
					f.Close()
				}
			}()
			if err := uncompress(ctx, r, tmpDir, cfg, tt.uncompress, tt.ext); err != nil {
				t.Errorf("%v: Unpack() error = %v", tt.name, err)
			}

			// check if file was extracted
			if _, err := os.Stat(filepath.Join(tmpDir, filename)); err != nil {
				t.Errorf("%v: File not found: %v", tt.name, err)
			}
		})
	}
}
