package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

func TestDecompress(t *testing.T) {

	ctx := context.Background()
	cfg := config.NewConfig()
	fileContent := []byte("Hello, World!")
	filename := "test"
	testTar := packTarWithContent([]tarContent{{Content: fileContent, Name: filename, Mode: 0640, Filetype: tar.TypeReg}})

	tests := []struct {
		name   string
		comp   compressFunc
		decomp decompressionFunction
		ext    string
	}{
		{
			name:   "gzip",
			comp:   compressGzip,
			decomp: decompressGZipStream,
			ext:    fileExtensionGZip,
		},
		{
			name:   "zstd",
			comp:   compressZstd,
			decomp: decompressZstdStream,
			ext:    fileExtensionZstd,
		},
		{
			name:   "bzip2",
			comp:   compressBzip2,
			decomp: decompressBz2Stream,
			ext:    fileExtensionBzip2,
		},
		{
			name:   "xz",
			comp:   compressXz,
			decomp: decompressXzStream,
			ext:    fileExtensionXz,
		},
		{
			name:   "brotli",
			comp:   compressBrotli,
			decomp: decompressBrotliStream,
			ext:    fileExtensionBrotli,
		},
		{
			name:   "lz4",
			comp:   compressLZ4,
			decomp: decompressLZ4Stream,
			ext:    fileExtensionLZ4,
		},
		{
			name:   "snappy",
			comp:   compressSnappy,
			decomp: decompressSnappyStream,
			ext:    fileExtensionSnappy,
		},
		{
			name:   "zlib",
			comp:   compressZlib,
			decomp: decompressZlibStream,
			ext:    fileExtensionZlib,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test.tar.%s", tt.ext))
			r := createFile(testFile, tt.comp(testTar))
			if err := decompress(ctx, r, tmpDir, cfg, tt.decomp, tt.ext); err != nil {
				t.Errorf("Unpack() error = %v", err)
			}

			// check if file was extracted
			if _, err := os.Stat(filepath.Join(tmpDir, filename)); err != nil {
				t.Errorf("File not found: %v", err)
			}
		})
	}
}

// compressFunc is a function that compresses a byte slice
type compressFunc func([]byte) []byte

// createCompressedTar creates a compressed tar file with the given data and provided compression function
func compressedTar(data []tarContent, compress compressFunc) []byte {
	return compress(packTarWithContent(data))
}
