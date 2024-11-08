package extract_test

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
)

// compressFunc is a function that compresses a byte slice
type compressFunc func(*testing.T, []byte) []byte

func TestDecompressTarCompress(t *testing.T) {
	ctx := context.Background()

	cfg := extract.NewConfig()

	filename := "test"

	testTar := packTar(t, []archiveContent{
		{
			Name:     filename,
			Content:  []byte("Hello, World!"),
			Mode:     0640,
			Filetype: tar.TypeReg,
		},
	})

	tests := []struct {
		name      string
		compress  compressFunc
		extension string
	}{
		{
			name:      "gzip",
			compress:  compressGzip,
			extension: extract.FileExtensionGZip,
		},
		{
			name:      "zstd",
			compress:  compressZstd,
			extension: extract.FileExtensionZstd,
		},
		{
			name:      "bzip2",
			compress:  compressBzip2,
			extension: extract.FileExtensionBzip2,
		},
		{
			name:      "xz",
			compress:  compressXz,
			extension: extract.FileExtensionXz,
		},
		{
			name:      "brotli",
			compress:  compressBrotli,
			extension: extract.FileExtensionBrotli,
		},
		{
			name:      "lz4",
			compress:  compressLZ4,
			extension: extract.FileExtensionLZ4,
		},
		{
			name:      "snappy",
			compress:  compressSnappy,
			extension: extract.FileExtensionSnappy,
		},
		{
			name:      "zlib",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test.tar.%s", test.extension))
			r := newTestFile(testFile, test.compress(t, testTar))
			defer func() {
				if f, ok := r.(io.Closer); ok {
					f.Close()
				}
			}()
			if err := extract.Unpack(ctx, r, tmpDir, cfg); err != nil {
				t.Errorf("%v: Unpack() error = %v", test.name, err)
			}

			// check if file was extracted
			if _, err := os.Stat(filepath.Join(tmpDir, filename)); err != nil {
				t.Errorf("%v: File not found: %v", test.name, err)
			}
		})
	}
}

func TestDecompressCompressedFile(t *testing.T) {
	ctx := context.Background()

	cfg := extract.NewConfig()

	fileContent := []byte("Hello, World!")

	filename := "test"

	tests := []struct {
		name      string
		dst       string
		cfg       *extract.Config
		compress  compressFunc
		extension string
		prep      func(*testing.T, string)
		outname   string
	}{
		{
			name:      "zlib",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
			dst:       "foo",
			outname:   "foo",
		},
		{
			name:      "zlib",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
			cfg:       extract.NewConfig(extract.WithCreateDestination(true)),
			dst:       "foo/bar",
			outname:   "foo/bar",
		},
		{
			name:      "zlib",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
			cfg:       extract.NewConfig(extract.WithCreateDestination(true)),
			dst:       "existing_dir",
			outname:   "existing_dir/test",
			prep: func(t *testing.T, tmpDir string) {
				if err := os.Mkdir(filepath.Join(tmpDir, "existing_dir"), 0755); err != nil {
					t.Fatalf("os.Mkdir() error = %v", err)
				}
			},
		},
		{
			name:      "zlib",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
			cfg:       extract.NewConfig(extract.WithOverwrite(true)),
			dst:       "existing_file",
			outname:   "existing_file",
			prep: func(t *testing.T, tmpDir string) {
				if err := os.WriteFile(filepath.Join(tmpDir, "existing_file"), fileContent, 0644); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
			},
		},
		{
			name:      "dst is link to existing folder",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
			dst:       "link_to_other_dir", // if dst is a symlink to a folder, the file should be extracted to the target of the symlink (bc/ dst is not sanitized)
			outname:   "link_to_other_dir/test",
			prep: func(t *testing.T, tmpDir string) {
				externalDir := t.TempDir()
				if err := os.Symlink(externalDir, filepath.Join(tmpDir, "link_to_other_dir")); err != nil {
					t.Fatalf("os.Symlink() error = %v", err)
				}
			},
		},
		{
			name:      "dst is link to existing file (WithOverwrite)",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
			cfg:       extract.NewConfig(extract.WithOverwrite(true)),
			dst:       "link_to_other_file", // if dst is a symlink to a file, the file should be overwritten (bc/ dst is not sanitized)
			outname:   "link_to_other_file",
			prep: func(t *testing.T, tmpDir string) {
				if err := os.WriteFile(filepath.Join(tmpDir, "existing_file"), fileContent, 0644); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
				if err := os.Symlink("existing_file", filepath.Join(tmpDir, "link_to_other_file")); err != nil {
					t.Fatalf("os.Symlink() error = %v", err)
				}
			},
		},
		{
			name:      "dst is link to non-existing file",
			compress:  compressZlib,
			extension: extract.FileExtensionZlib,
			dst:       "link_to_non_existing_file", // if dst is a symlink to a non-existing file, the file should be overwritten (bc/ dst is not sanitized)
			outname:   "link_to_non_existing_file",
			prep: func(t *testing.T, tmpDir string) {
				if err := os.Symlink("non_existing_file", filepath.Join(tmpDir, "link_to_non_existing_file")); err != nil {
					t.Fatalf("os.Symlink() error = %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.prep != nil {
				test.prep(t, tmpDir)
			}
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test.%s", test.extension))
			r := newTestFile(testFile, test.compress(t, fileContent))
			defer func() {
				if f, ok := r.(io.Closer); ok {
					f.Close()
				}
			}()
			if test.cfg == nil {
				test.cfg = cfg
			}
			dst := filepath.Join(tmpDir, test.dst)
			if err := extract.Unpack(ctx, r, dst, test.cfg); err != nil {
				t.Errorf("%v: Unpack() error = %v", test.name, err)
			}

			// check if file was extracted
			checkFile := filepath.Join(tmpDir, filename)
			if test.outname != "" {
				checkFile = filepath.Join(tmpDir, test.outname)
			}
			if _, err := os.Stat(checkFile); err != nil {
				t.Errorf("%v: File not found: %v", test.name, err)
			}
		})
	}
}

// func FuzzDetermineOutputName(f *testing.F) {
// 	corpus := []string{
// 		"test.gz",
// 		"test",
// 	}

// 	for _, input := range corpus {
// 		f.Add(input)
// 	}

// 	checkedNames := make(map[string]struct{})
// 	mu := &sync.Mutex{}

// 	// perform fuzzing test and ignore errors, looking for panics!
// 	f.Fuzz(func(t *testing.T, fName string) {

// 		// prepare tmp
// 		dest := t.TempDir()

// 		// fuzz function with random data
// 		dir, outputName := determineOutputName(testingTarget, dest, fName, ".gz")

// 		// check if outputName is already checked, then skip
// 		if _, ok := checkedNames[outputName]; ok {
// 			return
// 		}

// 		// lock and add outputName to checkedNames
// 		mu.Lock()
// 		checkedNames[outputName] = struct{}{}
// 		mu.Unlock()

// 		// write file to check if outputName is correct determined
// 		if err := os.WriteFile(filepath.Join(dir, outputName), []byte("Hello World!"), 0644); err != nil {
// 			t.Errorf("os.WriteFile() error = %v", err)
// 		}

// 	})
// }
