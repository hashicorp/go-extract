package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/hashicorp/go-extract/config"
)

// compressFunc is a function that compresses a byte slice
type compressFunc func([]byte) []byte

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
			ext:    FileExtensionGZip,
		},
		{
			name:   "zstd",
			comp:   compressZstd,
			decomp: decompressZstdStream,
			ext:    FileExtensionZstd,
		},
		{
			name:   "bzip2",
			comp:   compressBzip2,
			decomp: decompressBz2Stream,
			ext:    FileExtensionBzip2,
		},
		{
			name:   "xz",
			comp:   compressXz,
			decomp: decompressXzStream,
			ext:    FileExtensionXz,
		},
		{
			name:   "brotli",
			comp:   compressBrotli,
			decomp: decompressBrotliStream,
			ext:    FileExtensionBrotli,
		},
		{
			name:   "lz4",
			comp:   compressLZ4,
			decomp: decompressLZ4Stream,
			ext:    FileExtensionLZ4,
		},
		{
			name:   "snappy",
			comp:   compressSnappy,
			decomp: decompressSnappyStream,
			ext:    FileExtensionSnappy,
		},
		{
			name:   "zlib",
			comp:   compressZlib,
			decomp: decompressZlibStream,
			ext:    FileExtensionZlib,
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
			if err := decompress(ctx, r, tmpDir, cfg, tt.decomp, tt.ext); err != nil {
				t.Errorf("%v: Unpack() error = %v", tt.name, err)
			}

			// check if file was extracted
			if _, err := os.Stat(filepath.Join(tmpDir, filename)); err != nil {
				t.Errorf("%v: File not found: %v", tt.name, err)
			}
		})
	}
}

func FuzzDetermineOutputName(f *testing.F) {
	content := compressGzip([]byte("Hello, World!"))
	cases := []string{
		"test.gz",
		"test",
	}
	for _, tc := range cases {
		f.Add(tc)
	}

	// perform fuzzing test and ignore errors, looking for panics!
	cfg := config.NewConfig()
	f.Fuzz(func(t *testing.T, fName string) {

		// ensure that the filename is not a path
		fName = filepath.Base(fName)

		dest := t.TempDir()
		path := filepath.Join(dest, fName)
		if !utf8.ValidString(path) {
			// log.Fatalf("Invalid UTF-8 in path: %s", path)
			return
		}
		// check if path exists
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return // file can only be created if it does not exist and there are no other errors
		}
		// ignore errors, bc/ if input cannot exist, we do not need to test with this input
		if err := os.WriteFile(path, content, 0640); err != nil {
			// panic(fmt.Errorf("os.WriteFile() error = %v", err))
			return
		}
		fin, err := os.Open(path)
		if err != nil {
			return // ignore errors, bc/ file cannot be opened for some reason
		}
		defer fin.Close()
		ctx := context.Background()
		if err := decompress(ctx, fin, dest, cfg, decompressGZipStream, FileExtensionGZip); err != nil {
			t.Errorf("decompress() error = %v", err)
		}
	})
}
