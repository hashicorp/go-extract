package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// compressFunc is a function that compresses a byte slice
type compressFunc func([]byte) []byte

func TestDecompressTarCompress(t *testing.T) {

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

			// Create a new target
			testingTarget := target.NewOS()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test.tar.%s", tt.ext))
			r := newTestFile(testFile, tt.comp(testTar))
			defer func() {
				if f, ok := r.(io.Closer); ok {
					f.Close()
				}
			}()
			if err := decompress(ctx, testingTarget, tmpDir, r, cfg, tt.decomp, tt.ext); err != nil {
				t.Errorf("%v: Unpack() error = %v", tt.name, err)
			}

			// check if file was extracted
			if _, err := os.Stat(filepath.Join(tmpDir, filename)); err != nil {
				t.Errorf("%v: File not found: %v", tt.name, err)
			}
		})
	}
}

func TestDecompressCompressedFile(t *testing.T) {

	ctx := context.Background()
	cfg := config.NewConfig()
	fileContent := []byte("Hello, World!")
	filename := "test"

	tests := []struct {
		name    string
		dst     string
		cfg     *config.Config
		comp    compressFunc
		decomp  decompressionFunction
		ext     string
		prep    func(string)
		outname string
	}{
		{
			name:    "zlib",
			comp:    compressZlib,
			decomp:  decompressZlibStream,
			ext:     FileExtensionZlib,
			dst:     "foo",
			outname: "foo",
		},
		{
			name:    "zlib",
			comp:    compressZlib,
			decomp:  decompressZlibStream,
			ext:     FileExtensionZlib,
			cfg:     config.NewConfig(config.WithCreateDestination(true)),
			dst:     "foo/bar",
			outname: "foo/bar",
		},
		{
			name:    "zlib",
			comp:    compressZlib,
			decomp:  decompressZlibStream,
			ext:     FileExtensionZlib,
			cfg:     config.NewConfig(config.WithCreateDestination(true)),
			dst:     "existing_dir",
			outname: "existing_dir/test",
			prep: func(tmpDir string) {
				if err := os.Mkdir(filepath.Join(tmpDir, "existing_dir"), 0755); err != nil {
					t.Errorf("os.Mkdir() error = %v", err)
				}
			},
		},
		{
			name:    "zlib",
			comp:    compressZlib,
			decomp:  decompressZlibStream,
			ext:     FileExtensionZlib,
			cfg:     config.NewConfig(config.WithOverwrite(true)),
			dst:     "existing_file",
			outname: "existing_file",
			prep: func(tmpDir string) {
				if err := os.WriteFile(filepath.Join(tmpDir, "existing_file"), fileContent, 0644); err != nil {
					t.Errorf("os.WriteFile() error = %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create a new target
			testingTarget := target.NewOS()

			tmpDir := t.TempDir()
			if tt.prep != nil {
				tt.prep(tmpDir)
			}
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test.%s", tt.ext))
			r := newTestFile(testFile, tt.comp(fileContent))
			defer func() {
				if f, ok := r.(io.Closer); ok {
					f.Close()
				}
			}()
			if tt.cfg == nil {
				tt.cfg = cfg
			}
			dst := filepath.Join(tmpDir, tt.dst)
			if err := decompress(ctx, testingTarget, dst, r, tt.cfg, tt.decomp, tt.ext); err != nil {
				t.Errorf("%v: Unpack() error = %v", tt.name, err)
			}

			// check if file was extracted
			checkFile := filepath.Join(tmpDir, filename)
			if tt.outname != "" {
				checkFile = filepath.Join(tmpDir, tt.outname)
			}
			if _, err := os.Stat(checkFile); err != nil {
				t.Errorf("%v: File not found: %v", tt.name, err)
			}
		})
	}
}

func FuzzDetermineOutputName(f *testing.F) {
	cases := []string{
		"test.gz",
		"test",
	}
	for _, tc := range cases {
		f.Add(tc)
	}

	checkedNames := make(map[string]struct{})
	mu := &sync.Mutex{}

	// perform fuzzing test and ignore errors, looking for panics!
	f.Fuzz(func(t *testing.T, fName string) {

		// Create a new target
		testingTarget := target.NewOS()

		// prepare tmp
		dest := t.TempDir()

		// fuzz function with random data
		dir, outputName := determineOutputName(testingTarget, dest, fName, ".gz")

		// check if outputName is already checked, then skip
		if _, ok := checkedNames[outputName]; ok {
			return
		}

		// lock and add outputName to checkedNames
		mu.Lock()
		checkedNames[outputName] = struct{}{}
		mu.Unlock()

		// write file to check if outputName is correct determined
		if err := os.WriteFile(filepath.Join(dir, outputName), []byte("Hello World!"), 0644); err != nil {
			t.Errorf("os.WriteFile() error = %v", err)
		}

	})
}
