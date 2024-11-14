package extract_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/dsnet/compress/bzip2"
	"github.com/golang/snappy"
	"github.com/hashicorp/go-extract"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/ulikunitz/xz"
)

func ExampleUnpack() {
	var (
		ctx = context.Background()      // context for cancellation
		dst = createDirectory("output") // create destination directory
		src = openFile("example.zip")   // source reader
		cfg = extract.NewConfig()       // custom config for extraction
	)

	// unpack
	if err := extract.Unpack(ctx, src, dst, cfg); err != nil {
		// handle error
	}

	// read extracted file
	content, err := os.ReadFile(filepath.Join(dst, "example.txt"))
	if err != nil {
		// handle error
	}
	fmt.Println(string(content))

	// Output:
	// example content
}

func ExampleUnpackTo() {
	var (
		ctx = context.Background()      // context for cancellation
		m   = extract.NewMemoryTarget() // create a new in-memory filesystem
		dst = ""                        // root of in-memory filesystem
		src = openFile("example.zip")   // source reader
		cfg = extract.NewConfig()       // custom config for extraction
	)

	// unpack
	if err := extract.UnpackTo(ctx, m, dst, src, cfg); err != nil {
		// handle error
	}

	// read extracted file
	memFS := m.(fs.FS)
	content, err := fs.ReadFile(memFS, "example.txt")
	if err != nil {
		// handle error
	}
	fmt.Println(string(content))

	// Output:
	// example content
}

func ExampleNewMemoryTarget() {
	var (
		ctx = context.Background()      // context for cancellation
		m   = extract.NewMemoryTarget() // create a new in-memory filesystem
		dst = ""                        // root of in-memory filesystem
		src = openFile("example.zip")   // source reader
		cfg = extract.NewConfig()       // custom config for extraction
	)

	// unpack
	if err := extract.UnpackTo(ctx, m, dst, src, cfg); err != nil {
		// handle error
	}

	// Walk the memory filesystem
	memFs := m.(fs.FS)
	if err := fs.WalkDir(memFs, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		fmt.Println(path)
		return nil
	}); err != nil {
		fmt.Printf("failed to walk memory filesystem: %s", err)
		return
	}

	// Output:
	// example.txt
}

func ExampleNewDiskTarget() {
	var (
		ctx = context.Background()    // context for cancellation
		d   = extract.NewDiskTarget() // local filesystem
		dst = createDirectory("out")  // create destination directory
		src = openFile("example.zip") // source reader
		cfg = extract.NewConfig()     // custom config for extraction
	)

	// unpack
	if err := extract.UnpackTo(ctx, d, dst, src, cfg); err != nil {
		// handle error
	}

	// read extracted file
	content, err := os.ReadFile(filepath.Join(dst, "example.txt"))
	if err != nil {
		// handle error
	}
	fmt.Println(string(content))

	// Output:
	// example content
}

// Demonstrates how to extract an "example.zip" source archive to an "output" directory on
// disk with the default configuration options.
func Example() {
	var (
		ctx = context.Background()      // context for cancellation
		src = openFile("example.zip")   // source reader
		dst = createDirectory("output") // create destination directory
		cfg = extract.NewConfig()       // custom config for extraction
	)

	err := extract.Unpack(ctx, src, dst, cfg)
	if err != nil {
		switch {
		case errors.Is(err, extract.ErrNoExtractorFound):
			// handle no extractor found
		case errors.Is(err, extract.ErrUnsupportedFileType):
			// handle unsupported file type
		case errors.Is(err, extract.ErrFailedToReadHeader):
			// handle failed to read header
		case errors.Is(err, extract.ErrFailedToUnpack):
			// handle failed to unpack
		default:
			// handle other error
		}
	}

	content, err := os.ReadFile(filepath.Join(dst, "example.txt"))
	if err != nil {
		// handle error
	}

	fmt.Println(string(content))
	// Output: example content
}

func TestUnpackCompressed(t *testing.T) {

	tests := []struct {
		name       string
		compressor func(*testing.T, []byte) []byte
		ext        string
	}{
		{
			name:       "brotli",
			compressor: compressBrotli,
			ext:        "br",
		},
		{
			name:       "gzip",
			compressor: compressGzip,
			ext:        "gz",
		},
		{
			name:       "bzip2",
			compressor: compressBzip2,
			ext:        "bz2",
		},
		{
			name:       "lz4",
			compressor: compressLZ4,
			ext:        "lz4",
		},
		{
			name:       "snappy",
			compressor: compressSnappy,
			ext:        "sz",
		},
		{
			name:       "xz",
			compressor: compressXz,
			ext:        "xz",
		},
		{
			name:       "zlib",
			compressor: compressZlib,
			ext:        "zlib",
		},
		{
			name:       "zstd",
			compressor: compressZstd,
			ext:        "zst",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			var (
				tmp  = t.TempDir()
				data = []byte("test data")
				ctx  = context.Background()
				dst  = fmt.Sprintf("%v/decompressed", tmp)
				src  = createFileReader(t, fmt.Sprintf("*.%s", test.ext), test.compressor(t, data))
				cfg  = extract.NewConfig()
			)

			if err := extract.Unpack(ctx, src, dst, cfg); err != nil {
				t.Fatalf("[%s] error decompressing data: %v", test.name, err)
			}
			content, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("[%s] error reading decompressed file: %v", test.name, err)
			}
			if string(content) != string(data) {
				t.Fatalf("[%s] expected %s, got %s", test.name, data, content)
			}

		})
	}
}

func TestUnpackArchive(t *testing.T) {

	ta := []archiveContent{
		{
			Name:    "test",
			Content: []byte("hello world"),
			Mode:    0644,
		},
		{
			Name: "dir",
			Mode: fs.ModeDir | 0755,
		},
		{
			Name:    "dir/entry",
			Content: []byte("hello world"),
			Mode:    0644,
		},
		{
			Name:       "dir/link",
			Linktarget: "../test",
			Mode:       fs.ModeSymlink | 0755,
		},
	}

	testCases := []struct {
		name      string
		src       []byte
		noSymlink bool
	}{
		{
			name: "tar",
			src:  packTar(t, ta),
		},
		{
			name: "zip",
			src:  packZip(t, ta),
		},
		{
			name:      "7z",
			src:       pack7z(t, ta),
			noSymlink: true,
		},
		{
			name:      "rar",
			src:       packRar(t, ta),
			noSymlink: true,
		},
	}

	for _, tc := range testCases {
		for _, cacheFunction := range []func(*testing.T, []byte) io.Reader{asIoReader, asFileReader} {
			t.Run(tc.name, func(t *testing.T) {

				var (
					ctx = context.Background()
					dst = t.TempDir()
					src = cacheFunction(t, tc.src)
					cfg = extract.NewConfig(extract.WithCreateDestination(true), extract.WithContinueOnUnsupportedFiles(true))
				)

				if err := extract.Unpack(ctx, src, dst, cfg); err != nil {
					t.Fatalf("[%s] error extracting data: %v", tc.name, err)
				}

				for _, c := range ta {
					if tc.noSymlink && c.Mode&fs.ModeSymlink != 0 {
						continue // skip symlink test
					}
					path := filepath.Join(dst, c.Name)
					fi, err := os.Lstat(path)
					if err != nil {
						t.Fatalf("[%s] error stating file: %v", tc.name, err)
					}
					if c.Mode.IsDir() && !fi.IsDir() {
						t.Fatalf("[%s] expected directory, got file", tc.name)
					}
					if c.Mode&fs.ModeSymlink != 0 && fi.Mode()&fs.ModeSymlink == 0 {
						t.Fatalf("[%s] expected symlink, got file", tc.name)
					}
					if c.Mode.IsRegular() && !fi.Mode().IsRegular() {
						t.Fatalf("[%s] expected regular file, got directory: %s", tc.name, c.Name)
					}
				}
			})
		}
	}

}

func TestUnpackMaliciousArchive(t *testing.T) {

	var testCases = []struct {
		name        string
		entries     []archiveContent
		expectError bool
	}{
		{
			name: "single file",
			entries: []archiveContent{
				{Name: "test", Mode: 0640, Content: []byte("foobar content")},
			},
		},
		{
			name: "path traversal in name",
			entries: []archiveContent{
				{Name: "../escaped", Mode: 0640, Content: []byte("foobar content")},
			},
			expectError: true,
		},
		{
			name: "path traversal in name, but thats okay, bc/ its in a sub directory",
			entries: []archiveContent{
				{Name: "sub/../ok", Mode: 0640, Content: []byte("foobar content")},
			},
		},
		{
			name: "symlink to outside",
			entries: []archiveContent{
				{Name: "outside", Mode: fs.ModeSymlink | 0755, Linktarget: "../"},
			},
			expectError: true,
		},
		{
			name: "symlink to absolute path",
			entries: []archiveContent{
				{Name: "etc-passwd", Mode: fs.ModeSymlink | 0755, Linktarget: "/etc/passwd"},
			},
			expectError: runtime.GOOS != "windows", // on windows, this is not an error
		},
		{
			name: "symlink with path traversal in name",
			entries: []archiveContent{
				{Name: "../escaped", Mode: fs.ModeSymlink | 0755, Linktarget: "fooo"},
			},
			expectError: true,
		},
		{
			name: "directory with path traversal in name",
			entries: []archiveContent{
				{Name: "../escaped", Mode: fs.ModeDir | 0755},
			},
			expectError: true,
		},
		{
			name: "zip-slip attack",
			entries: []archiveContent{
				{Name: "sub", Mode: fs.ModeDir | 0755},
				{Name: "sub/root", Mode: fs.ModeSymlink | 0755, Linktarget: "../"},
				{Name: "sub/root/one-above", Mode: fs.ModeSymlink | 0755, Linktarget: "../"},
				{Name: "sub/root/one-above/escaped", Mode: 0640, Content: []byte("foobar content")},
			},
			expectError: true,
		},
		{
			name: "zip-slip attack sneaky",
			entries: []archiveContent{
				{Name: "sub", Mode: fs.ModeDir | 0755},
				{Name: "sub/root", Mode: fs.ModeSymlink | 0755, Linktarget: "../"},
				{Name: "sub/root/one-above", Mode: fs.ModeSymlink | 0755, Linktarget: "../"},
				{Name: "sub/does-not-exist/../root/one-above/escaped", Mode: 0640, Content: []byte("foobar content")},
			},
			expectError: true,
		},
		{
			name: "malicious tar with file named '.'",
			entries: []archiveContent{
				{Name: ".", Mode: 0640, Content: []byte("foobar content")},
			},
			expectError: true,
		},
		{
			name: "malicious tar with file named '..'",
			entries: []archiveContent{
				{Name: "..", Mode: 0640, Content: []byte("foobar content")},
			},
			expectError: true,
		},
		{
			name: "absolute path in filename (windows)",
			entries: []archiveContent{
				{Name: "s:\\absolute-path", Mode: 0640, Content: []byte("foobar content")},
			},
			expectError: runtime.GOOS == "windows",
		},
		{
			name: "absolute path in link target (windows)",
			entries: []archiveContent{
				{Name: "test", Mode: fs.ModeSymlink | 0755, Linktarget: "s:\\absolute-path"},
			},
			expectError: runtime.GOOS == "windows",
		},
		{
			name: "link-writer attack",
			entries: []archiveContent{
				{Name: "test", Mode: fs.ModeSymlink | 0755, Linktarget: "../escaped"},
				{Name: "test", Mode: 0640, Content: []byte("foobar content")},
			},
			expectError: true,
		},
		{
			name: "link-chain attack",
			entries: []archiveContent{
				{Name: "sub", Mode: fs.ModeDir | 0755},
				{Name: "sub/escaped", Mode: fs.ModeSymlink | 0755, Linktarget: "../escaped"},
				{Name: "sub/escaped", Mode: fs.ModeSymlink | 0755, Linktarget: "../escaped"},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// prepare test
			var (
				ctx = context.Background()
				dst = t.TempDir()
				src = asIoReader(t, packTar(t, tc.entries))
				cfg = extract.NewConfig()
			)

			// perform test
			err := extract.Unpack(ctx, src, dst, cfg)

			// check if we got the expected error
			if tc.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestZipUnpackIllegalNames tests, with various cases, the implementation of zip.Unpack
func TestUnpackWithIllegalNames(t *testing.T) {

	// reserved names and forbidden characters
	// from: https://go.googlesource.com/go/+/refs/tags/go1.19.1/src/path/filepath/path_windows.go#19
	// from: https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names
	// removed `/` and `\` from tests, bc/ the zip lib cannot create directories as test file
	var reservedNames []string
	var forbiddenCharacters []string
	if runtime.GOOS == "windows" {
		reservedNames = []string{
			"CON", "PRN", "AUX", "NUL",
			"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		}
		forbiddenCharacters = []string{`<`, `>`, `:`, `"`, `|`, `?`, `*`}
		for i := 0; i <= 31; i++ {
			fmt.Println(string(byte(i)))
			forbiddenCharacters = append(forbiddenCharacters, string(byte(i)))
		}
	} else {
		forbiddenCharacters = []string{"\x00"}
	}
	testCases := append(reservedNames, forbiddenCharacters...)

	// test reserved names and forbidden chars
	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			var (
				ctx = context.Background()
				dst = t.TempDir()
				src = packZip(t, []archiveContent{
					{Name: tc, Content: []byte("hello world"), Mode: 0644},
				})
			)
			if err := extract.Unpack(ctx, asIoReader(t, src), dst, extract.NewConfig()); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestUnpackWithConfig(t *testing.T) {

	defaultArchive := []archiveContent{
		{
			Name:    "test",
			Content: []byte("hello world"),
			Mode:    0644,
		},
		{
			Name: "dir",
			Mode: fs.ModeDir | 0755,
		},
		{
			Name:       "dir/link",
			Linktarget: "../test",
			Mode:       fs.ModeSymlink | 0755,
		},
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := []struct {
		name        string
		cfg         *extract.Config
		ctx         context.Context
		testArchive []archiveContent
		dst         string
		expectError bool
	}{
		{
			name: "unpack normal",
			cfg:  extract.NewConfig(),
		},
		{
			name: "unpack with destination",
			cfg:  extract.NewConfig(extract.WithCreateDestination(true)),
			dst:  "sub",
		},
		{
			name: "unpack with pattern missmatch",
			cfg:  extract.NewConfig(extract.WithPatterns("*foo*")),
		},
		{
			name:        "unpack with canceled context",
			ctx:         canceledCtx,
			expectError: true,
		},
		{
			name:        "unpack with file limit",
			cfg:         extract.NewConfig(extract.WithMaxFiles(2)),
			expectError: true,
		},
		{
			name: "unpack with file cache in memory",
			cfg:  extract.NewConfig(extract.WithCacheInMemory(true)),
		},
		{
			name:        "unpack with max input size",
			cfg:         extract.NewConfig(extract.WithMaxInputSize(1)),
			expectError: true,
		},
		{
			name:        "archive with windows paths",
			testArchive: []archiveContent{{Name: `example-dir\foo\bar\test`, Content: []byte("hello world"), Mode: 0644}},
		},
		{
			name:        "unpack with extraction size limit",
			cfg:         extract.NewConfig(extract.WithMaxExtractionSize(1)),
			expectError: true,
		},
		{
			name:        "unpack with continue on error",
			cfg:         extract.NewConfig(extract.WithContinueOnError(true)),
			testArchive: []archiveContent{{Name: "../test", Content: []byte("hello world"), Mode: 0644}},
		},
		{
			name:        "unpack with deny symlink",
			cfg:         extract.NewConfig(extract.WithDenySymlinkExtraction(true)),
			expectError: true,
		},
		{
			name: "unpack with deny symlink and continue on error",
			cfg:  extract.NewConfig(extract.WithDenySymlinkExtraction(true), extract.WithContinueOnError(true)),
		},
		{
			name: "unpack with deny symlink and continue on unsupported files",
			cfg:  extract.NewConfig(extract.WithDenySymlinkExtraction(true), extract.WithContinueOnUnsupportedFiles(true)),
		},
		{
			name:        "unpack fifo",
			testArchive: []archiveContent{{Name: "../test", Content: []byte("hello world"), Mode: fs.ModeNamedPipe | 0755}},
			expectError: true,
		},
		{
			name: "tar with legit git pax_global_header",
			testArchive: []archiveContent{
				{Name: "pax_global_header", Mode: fs.FileMode(tar.TypeXGlobalHeader)},
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
			},
		},
		{
			name: "unpack with  overwrite disabled",
			testArchive: []archiveContent{
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
			},
			expectError: true,
		},
		{
			name: "unpack with overwrite enabled",
			testArchive: []archiveContent{
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
			},
			cfg:         extract.NewConfig(extract.WithOverwrite(true)),
			expectError: false,
		},
		{
			name: "traverse symlink disabled",
			testArchive: []archiveContent{
				{Name: "dir", Mode: fs.ModeDir | 0755},
				{Name: "link", Mode: fs.ModeSymlink | 0755, Linktarget: "dir"},
				{Name: "link/test", Content: []byte("hello world"), Mode: 0644},
			},
			expectError: true,
		},
		{
			name: "traverse symlink enabled",
			testArchive: []archiveContent{
				{Name: "dir", Mode: fs.ModeDir | 0755},
				{Name: "link", Mode: fs.ModeSymlink | 0755, Linktarget: "dir"},
				{Name: "link/test", Content: []byte("hello world"), Mode: 0644},
			},
			cfg:         extract.NewConfig(extract.WithFollowSymlinks(true)),
			expectError: false,
		},
	}

	packer := []struct {
		name string
		pack func(*testing.T, []archiveContent) []byte
	}{
		{
			name: "tar",
			pack: packTar,
		},
		{
			name: "zip",
			pack: packZip,
		},
	}

	for _, tc := range testCases {
		for _, p := range packer {
			t.Run(tc.name, func(t *testing.T) {

				if tc.ctx == nil {
					tc.ctx = context.Background()
				}
				if tc.testArchive == nil {
					tc.testArchive = defaultArchive
				}
				if tc.cfg == nil {
					tc.cfg = extract.NewConfig()
				}

				var (
					ctx = tc.ctx
					tmp = t.TempDir()
					src = createFileReader(t, fmt.Sprintf("*.%s", p.name), p.pack(t, tc.testArchive))
					dst = filepath.Join(tmp, tc.dst)
					cfg = tc.cfg
				)

				err := extract.Unpack(ctx, src, dst, cfg)

				if tc.expectError && err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !tc.expectError && err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	}
}

func TestDecompression(t *testing.T) {

	// 1024 * A
	defaultContent := bytes.Repeat([]byte("A"), 1024)
	compressed := compressZlib(t, defaultContent)
	exampleTarGz := compressGzip(t, packTar(t, []archiveContent{{Name: "test", Content: defaultContent, Mode: 0644}}))
	outputName := "decompressed"

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := []struct {
		name        string
		src         io.Reader
		cfg         *extract.Config
		ctx         context.Context
		expectError bool
	}{
		{
			name: "normal decompression",
			src:  asIoReader(t, compressed),
		},
		{
			name:        "decompression with canceled context",
			src:         asIoReader(t, compressed),
			ctx:         cancelCtx,
			expectError: true,
		},
		{
			name:        "decompression with max input size",
			src:         asIoReader(t, compressed),
			cfg:         extract.NewConfig(extract.WithMaxInputSize(1)),
			expectError: true,
		},
		{
			name:        "decompression with max extraction size",
			src:         asIoReader(t, compressed),
			cfg:         extract.NewConfig(extract.WithMaxExtractionSize(1)),
			expectError: true,
		},
		{
			name: "extract after decompression true",
			src:  asIoReader(t, exampleTarGz),
			cfg:  extract.NewConfig(extract.WithCreateDestination(true)),
		},
		{
			name: "extract after decompression false",
			src:  asIoReader(t, exampleTarGz),
			cfg:  extract.NewConfig(extract.WithNoUntarAfterDecompression(true)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			if tc.ctx == nil {
				tc.ctx = context.Background()
			}
			if tc.cfg == nil {
				tc.cfg = extract.NewConfig()
			}

			var (
				ctx = tc.ctx
				tmp = t.TempDir()
				dst = filepath.Join(tmp, outputName)
				cfg = tc.cfg
			)

			err := extract.Unpack(ctx, tc.src, dst, cfg)

			if tc.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}

}

func compressBrotli(t *testing.T, data []byte) []byte {
	t.Helper()
	b := new(bytes.Buffer)
	w := brotli.NewWriter(b)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to brotli writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing brotli writer: %v", err)
	}
	return b.Bytes()
}

func compressGzip(t *testing.T, data []byte) []byte {
	t.Helper()
	b := new(bytes.Buffer)
	w := gzip.NewWriter(b)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to gzip writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing gzip writer: %v", err)
	}
	return b.Bytes()
}

// compressBzip2 compresses data with bzip2 algorithm.
func compressBzip2(t *testing.T, data []byte) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	w, err := bzip2.NewWriter(buf, &bzip2.WriterConfig{
		Level: bzip2.DefaultCompression,
	})
	if err != nil {
		t.Fatalf("error creating bzip2 writer: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to bzip2 writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing bzip2 writer: %v", err)
	}
	return buf.Bytes()
}

func compressLZ4(t *testing.T, data []byte) []byte {
	t.Helper()
	b := new(bytes.Buffer)
	w := lz4.NewWriter(b)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to lz4 writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing lz4 writer: %v", err)
	}
	return b.Bytes()
}

func compressSnappy(t *testing.T, data []byte) []byte {
	t.Helper()
	b := new(bytes.Buffer)
	w := snappy.NewBufferedWriter(b)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to snappy writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing snappy writer: %v", err)
	}
	return b.Bytes()
}

func compressXz(t *testing.T, data []byte) []byte {
	t.Helper()
	b := new(bytes.Buffer)
	w, err := xz.NewWriter(b)
	if err != nil {
		t.Fatalf("error creating xz writer: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to xz writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing xz writer: %v", err)
	}
	return b.Bytes()
}

func compressZlib(t *testing.T, data []byte) []byte {
	t.Helper()
	b := new(bytes.Buffer)
	w := zlib.NewWriter(b)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to zlib writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing zlib writer: %v", err)
	}
	return b.Bytes()
}

func compressZstd(t *testing.T, data []byte) []byte {
	t.Helper()
	b := new(bytes.Buffer)
	w, err := zstd.NewWriter(b, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		t.Fatalf("error creating zstd writer: %v", err)
	}

	if _, err := w.Write(data); err != nil {
		t.Fatalf("error writing data to zstd writer: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("error closing zstd writer: %v", err)
	}

	return b.Bytes()
}

// archiveContent is a struct to store the content of a file inside an archive
type archiveContent struct {
	Name       string
	Content    []byte
	Linktarget string
	Mode       fs.FileMode
}

// packTar creates a tar file with the given content
func packTar(t *testing.T, content []archiveContent) []byte {
	// t.Helper()
	b := bytes.NewBuffer([]byte{})
	w := tar.NewWriter(b)
	for _, c := range content {
		var tFlag byte
		switch {
		case c.Mode.IsDir():
			tFlag = tar.TypeDir
		case c.Mode&fs.ModeSymlink != 0:
			tFlag = tar.TypeSymlink
		case c.Mode == tar.TypeXGlobalHeader:
			tFlag = tar.TypeXGlobalHeader
		case c.Mode.IsRegular():
			tFlag = tar.TypeReg
		case c.Mode&fs.ModeNamedPipe != 0:
			tFlag = tar.TypeFifo
		case c.Mode&fs.ModeCharDevice != 0:
			tFlag = tar.TypeChar
		case c.Mode&fs.ModeDevice != 0:
			tFlag = tar.TypeBlock
		default:
			t.Fatalf("unsupported file mode: %v", c.Mode)
		}
		header := &tar.Header{
			Name:     c.Name,
			Mode:     int64(c.Mode & fs.ModePerm),
			Size:     int64(len(c.Content)),
			Linkname: c.Linktarget,
			Typeflag: tFlag,
		}
		if tFlag == tar.TypeXGlobalHeader {
			header.Mode = 0
			header.Size = 0
			header.Format = tar.FormatPAX
			header.PAXRecords = map[string]string{}
			header.PAXRecords["path"] = c.Name
		}
		if err := w.WriteHeader(header); err != nil {
			t.Fatalf("error writing tar header: %v", err)
		}
		if !c.Mode.IsRegular() {
			continue
		}
		if _, err := w.Write(c.Content); err != nil {
			t.Fatalf("error writing tar data: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing tar writer: %v", err)
	}
	return b.Bytes()
}

func packZip(t *testing.T, content []archiveContent) []byte {
	b := new(bytes.Buffer)
	w := zip.NewWriter(b)
	for _, c := range content {
		h := &zip.FileHeader{
			Name: c.Name,
		}
		h.SetMode(c.Mode)
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatalf("error creating zip header: %v", err)
		}
		if c.Mode&fs.ModeSymlink != 0 {
			if _, err := f.Write([]byte(c.Linktarget)); err != nil {
				t.Fatalf("error writing zip data: %v", err)
			}
		} else {
			if _, err := f.Write(c.Content); err != nil {
				t.Fatalf("error writing zip data: %v", err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing zip writer: %v", err)
	}
	return b.Bytes()
}

// pack7z creates always the same a 7z archive with following files:
// - dir			<- directory
// - test			<- file with content 'hello world'
// - dir/entry		<- file with content 'hello world'
// - dir/link		<- symlink to ../test
func pack7z(t *testing.T, _ []archiveContent) []byte {
	t.Helper()
	b, err := hex.DecodeString("377abcaf271c0004c56aaa05aa0000000000000022000000000000006f8f4694e0001e00195d00341949ee8de917893a335ffcaddde25ddffcba68ee826f0000000000813307ae0fd01dd27c9f3f47412d1ea0d6499572eff9701b44818f17d1ebf97a30988cb480987d5533695021ec7e826d40e780f3cc2281aa4269a8a6a4ca37325ce8144d61a65483cfaf19d952c49c1a6b394c806a28dea4123077df58998b710e178eaba4e90f9e59bc7e542d862968c5002d7b21b837330a6f57a080e68a0f5f3f38675600001706210109808900070b01000123030101055d001000000c80b60a015e606c030000")
	if err != nil {
		t.Fatalf("error decoding 7z data: %v", err)
	}
	return b
}

// packRar creates always the same a rar archive with following files:
// - dir			<- directory
// - test			<- file with content 'hello world'
// - dir/entry		<- file with content 'hello world'
// - dir/link		<- symlink to ../test
func packRar(t *testing.T, _ []archiveContent) []byte {
	t.Helper()
	b, err := hex.DecodeString("526172211a0701003392b5e50a01050600050101808000e371be362202030b8c00048c00a483022d3b08af80000104746573740a03136efb3167e4a0682868656c6c6f20776f726c640adcb502882702030b8c00048c00a483022d3b08af800001096469722f656e7472790a0313b7fc31670b0c701768656c6c6f20776f726c640ad4e90fbc30020317000407edc30200000000800001086469722f6c696e6b0a031386fb3167644557330b050100072e2e2f74657374d8f240b61b02030b000100ed8301800001036469720a03131f033267492769271d77565103050400")
	if err != nil {
		t.Fatalf("error decoding rar data: %v", err)
	}
	return b
}

// openFile is a helper function to "open" a file,
// but it returns an in-memory reader for example purposes.
func openFile(_ string) io.ReadCloser {
	b := bytes.NewBuffer(nil)

	zw := zip.NewWriter(b)

	f, err := zw.Create("example.txt")
	if err != nil {
		panic(err)
	}

	_, err = f.Write([]byte("example content"))
	if err != nil {
		panic(err)
	}

	if err := zw.Close(); err != nil {
		panic(err)
	}

	return io.NopCloser(b)
}

func createDirectory(name string) string {
	path, err := os.MkdirTemp(os.TempDir(), name)
	if err != nil {
		panic(err)
	}

	return path
}

func createFileReader(t *testing.T, name string, content []byte) io.Reader {
	t.Helper()
	f, err := os.CreateTemp("", name)
	if err != nil {
		t.Fatalf("error creating file: %v", err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatalf("error writing data: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("error seeking file: %v", err)
	}
	return f
}

func asFileReader(t *testing.T, b []byte) io.Reader {
	t.Helper()
	return createFileReader(t, "test*", b)
}

func asIoReader(t *testing.T, b []byte) io.Reader {
	t.Helper()
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		w.Write(b)
	}()
	return r
}

// createTestGzipWithFile creates a test gzip file in dstDir for testing
func createTestGzipWithFile(t *testing.T, dstDir string) string {
	p := compressGzip(t, []byte("foobar content"))
	targetFile := filepath.Join(dstDir, "GzipWithFile.gz")
	if err := os.WriteFile(targetFile, p, 0644); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// createTestZip is a helper function to generate test data
func createTestZip(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "TestZip.zip")

	// create a temporary dir for files in zip archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	archive, _ := os.Create(targetFile)
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	// prepare testfile for be added to zip
	testFilePath := filepath.Join(tmpDir, "test")
	createTestFile(t, testFilePath, "foobar content")
	f1, err := os.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	// write file into zip
	w1, _ := zipWriter.Create("test")
	if _, err := io.Copy(w1, f1); err != nil {
		t.Fatal(err)
	}

	// return path to zip
	return targetFile
}

// createTestNonArchive is a helper function to generate test data
func createTestNonArchive(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "test.txt")
	createTestFile(t, targetFile, "foo bar test")
	return targetFile
}

// createTestFile is a helper function to generate test files
func createTestFile(t *testing.T, path string, content string) {
	t.Helper()
	err := createTestFileWithPerm(path, content, 0640)
	if err != nil {
		t.Fatal(err)
	}
}

// createTestTar is a helper function to generate test data
func createTestTar(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "TarNormal.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer

	f, _ := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	tarWriter := tar.NewWriter(f)
	defer f.Close()

	// prepare testfile for be added to tar
	testFilePath := filepath.Join(tmpDir, "test")
	createTestFile(t, testFilePath, "foobar content")
	f1, err := os.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, filepath.Base(f1.Name()), f1)
	if err := addDirToTarArchive(tarWriter, "dir"); err != nil {
		t.Fatal(err)
	}

	// close tar
	tarWriter.Close()

	// return path to tar
	return targetFile
}

func addDirToTarArchive(tarWriter *tar.Writer, dirName string) error {
	header := &tar.Header{
		Name:     dirName,
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	return nil
}

func createTestTarWithFiles(t *testing.T, dst string, files map[string]string) {
	// prepare generated zip+writer
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tarWriter := tar.NewWriter(f)

	for nameInArchive, origFile := range files {
		f1, err := os.Open(origFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f1.Close()

		addFileToTarArchive(tarWriter, nameInArchive, f1)
	}

	err = tarWriter.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func addFileToTarArchive(tarWriter *tar.Writer, fileName string, f1 *os.File) {
	fileInfo, err := os.Lstat(f1.Name())
	if err != nil {
		panic(err)
	}

	// create a new dir/file header
	header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
	if err != nil {
		panic(err)
	}

	// adjust filename
	header.Name = fileName

	// write the header
	if err := tarWriter.WriteHeader(header); err != nil {
		panic(err)
	}

	// add content
	if _, err := io.Copy(tarWriter, f1); err != nil {
		panic(err)
	}
}

// TestUnpack is a test function
func TestUnpackToMemory(t *testing.T) {
	tests := []struct {
		name        string
		fn          func(*testing.T, string) string
		expectError bool
	}{
		{
			name:        "get zip extractor from file",
			fn:          createTestZip,
			expectError: false,
		},
		{
			name:        "get tar extractor from file",
			fn:          createTestTar,
			expectError: false,
		},
		{
			name:        "get gzip extractor from file",
			fn:          createTestGzipWithFile,
			expectError: false,
		},
		{
			name:        "get nil extractor fot textfile",
			fn:          createTestNonArchive,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// prepare vars
			want := test.expectError

			// perform actual tests
			archive, err := os.Open(test.fn(t, testDir))
			if err != nil {
				panic(err)
			}
			defer archive.Close()
			err = extract.UnpackTo(
				context.Background(),
				extract.NewMemory(),
				"",
				archive,
				extract.NewConfig(
					extract.WithOverwrite(true),
				),
			)
			got := err != nil

			// success if both are nil and no engine found
			if want != got {
				t.Errorf("\nexpected error: %v\ngot: %s\n", want, err)
			}
		})
	}
}

func gen1024ByteGzip(t *testing.T, dstDir string) string {
	testFile := filepath.Join(dstDir, "GzipWithFile.gz")
	b := compressGzip(t, []byte(strings.Repeat("A", 1024)))
	if err := os.WriteFile(testFile, b, 0644); err != nil {
		t.Fatal(err)
	}
	return testFile
}

func genSingleFileTar(t *testing.T, dstDir string) string {
	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// create test file
	testFile := filepath.Join(tmpDir, "testFile")
	createTestFile(t, testFile, strings.Repeat("A", 1024))

	tarFileName := filepath.Join(dstDir, "TarNormalSingleFile.tar")
	createTestTarWithFiles(t, tarFileName, map[string]string{"TestFile": testFile})
	return tarFileName
}

func genTarGzWith5Files(t *testing.T, dstDir string) string {
	var ac []archiveContent
	for i := 0; i < 5; i++ {
		ac = append(ac, archiveContent{
			Name:    fmt.Sprintf("testFile%d", i),
			Content: []byte(strings.Repeat("A", 1024)),
			Mode:    0644,
		})
	}
	tarByte := packTar(t, ac)
	gzByte := compressGzip(t, tarByte)
	tarGzFileName := filepath.Join(dstDir, "TarGzWith5Files.tar.gz")
	if err := os.WriteFile(tarGzFileName, gzByte, 0644); err != nil {
		t.Fatal(err)
	}
	return tarGzFileName
}

func TestTelemetryHook(t *testing.T) {
	tests := []struct {
		name                  string
		inputGenerator        func(*testing.T, string) string
		inputName             string
		dst                   string
		WithContinueOnError   bool
		WithCreateDestination bool
		WithMaxExtractionSize int64
		WithMaxFiles          int64
		WithOverwrite         bool
		expectedTelemetryData extract.TelemetryData
		expectError           bool
	}{
		{
			name:                  "normal gzip with file",
			inputGenerator:        gen1024ByteGzip,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gz",
			},
			expectError: false,
		},
		{
			name:                  "normal gzip with file, and decompression target-name",
			inputGenerator:        gen1024ByteGzip,
			dst:                   "target-file", // important: the gzip decompression has a filename das dst
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gz",
			},
			expectError: false,
		},
		{
			name:                  "normal gzip with file and decompression target-name in sub-dir failing",
			inputGenerator:        gen1024ByteGzip,
			inputName:             "GzipWithFile.gz",
			dst:                   "sub/target", // important: the gzip decompression has a filename as dst
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 1,
				ExtractionSize:   0,
				ExtractedType:    "gz",
			},
			expectError: true,
		},
		{
			name:                  "normal gzip with file, and decompression target-name in sub-dir with sub-dir-creation",
			inputGenerator:        gen1024ByteGzip,
			inputName:             "GzipWithFile.gz",
			dst:                   "sub/target", // important: the gzip decompression has a filename das dst
			WithContinueOnError:   false,
			WithCreateDestination: true,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gz",
			},
			expectError: false,
		},
		{
			name:                  "normal tar with file",
			inputGenerator:        genSingleFileTar,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "tar",
			},
			expectError: false,
		},
		{
			name:                  "normal tar with file, extracted file too big",
			inputGenerator:        genSingleFileTar,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1023,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 1,
				ExtractionSize:   0,
				ExtractedType:    "tar",
			},
			expectError: true,
		},
		{
			name:                  "normal tar.gz with 5 files",
			inputGenerator:        genTarGzWith5Files,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the size(tar-archive) > 1025 * 5
			WithMaxFiles:          5,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   5,
				ExtractionErrors: 0,
				ExtractionSize:   1024 * 5,
				ExtractedType:    "tar.gz",
			},
			expectError: false,
		},
		{
			name:                  "normal tar.gz with file with max files limit",
			inputGenerator:        genTarGzWith5Files,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the size(tar-archive) > 1025 * 5
			WithMaxFiles:          4,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   4,
				ExtractionErrors: 1,
				ExtractionSize:   1024 * 4,
				ExtractedType:    "tar.gz",
			},
			expectError: true,
		},
		{
			name:                  "normal tar.gz with file failing bc/ of missing sub directory",
			inputGenerator:        genTarGzWith5Files,
			dst:                   "sub",
			WithContinueOnError:   true,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the size(tar-archive) > 1025 * 5
			WithMaxFiles:          5,
			WithOverwrite:         false,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 5,
				ExtractionSize:   0,
				ExtractedType:    "tar.gz",
			},
			expectError: false,
		},
		{
			name:                  "normal zip file",
			inputGenerator:        createTestZip,
			dst:                   ".",
			WithMaxFiles:          1,
			WithMaxExtractionSize: 14,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   14,
				ExtractedType:    "zip",
			},
			expectError: false,
		},
		{
			name:                  "normal zip file extraction size exceeded",
			inputGenerator:        createTestZip,
			dst:                   ".",
			WithMaxExtractionSize: 10,
			expectedTelemetryData: extract.TelemetryData{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 1,
				ExtractionSize:   0,
				ExtractedType:    "zip",
			},
			expectError: true,
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// open file
			archive, err := os.Open(tc.inputGenerator(t, testDir))
			if err != nil {
				t.Fatal(err)
			}

			// prepare config
			var td *extract.TelemetryData
			hook := func(ctx context.Context, d *extract.TelemetryData) {
				td = d
			}

			cfg := extract.NewConfig(
				extract.WithContinueOnError(tc.WithContinueOnError),
				extract.WithCreateDestination(tc.WithCreateDestination),
				extract.WithMaxExtractionSize(tc.WithMaxExtractionSize),
				extract.WithMaxFiles(tc.WithMaxFiles),
				extract.WithOverwrite(tc.WithOverwrite),
				extract.WithTelemetryHook(hook),
			)

			// perform actual tests
			ctx := context.Background()
			dstDir := filepath.Join(testDir, tc.dst)
			err = extract.Unpack(ctx, archive, dstDir, cfg)
			archive.Close()

			// check if error is expected
			if tc.expectError != (err != nil) {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, tc.expectError, err)
			}

			// compare collected and expected ExtractedFiles
			if td.ExtractedFiles != tc.expectedTelemetryData.ExtractedFiles {
				t.Errorf("test case %d failed: %s (ExtractedFiles)\nexpected: %v\ngot: %v\n%v", i, tc.name, tc.expectedTelemetryData.ExtractedFiles, td.ExtractedFiles, td)
			}

			// compare collected and expected ExtractionErrors
			if td.ExtractionErrors != tc.expectedTelemetryData.ExtractionErrors {
				t.Errorf("test case %d failed: %s (ExtractionErrors)\nexpected: %v\ngot: %v", i, tc.name, tc.expectedTelemetryData.ExtractionErrors, td.ExtractionErrors)
			}

			// compare collected and expected ExtractionSize
			if td.ExtractionSize != tc.expectedTelemetryData.ExtractionSize {
				t.Errorf("test case %d failed: %s (ExtractionSize [e:%v|g:%v])\nexpected: %v\ngot: %v", i, tc.name, tc.expectedTelemetryData.ExtractionSize, td.ExtractionSize, tc.expectedTelemetryData.ExtractionSize, td.ExtractionSize)
			}

			// compare collected and expected ExtractedType
			if td.ExtractedType != tc.expectedTelemetryData.ExtractedType {
				t.Errorf("test case %d failed: %s (ExtractedType)\nexpected: %v\ngot: %v", i, tc.name, tc.expectedTelemetryData.ExtractedType, td.ExtractedType)
			}

		})
	}
}

func TestUnpackWithTypes(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *extract.Config
		archiveName   string
		content       []byte
		gen           func(target string, data []byte) io.Reader
		expectedFiles []string
		expectError   bool
	}{
		{
			name:          "get zip extractor from file",
			cfg:           extract.NewConfig(extract.WithExtractType(extract.FileExtensionGZip)),
			archiveName:   "TestZip.gz",
			content:       compressGzip(t, []byte("foobar content")),
			gen:           createFile,
			expectedFiles: []string{"TestZip"},
			expectError:   false,
		},
		{
			name:        "set type to non-valid type and expect error",
			cfg:         extract.NewConfig(extract.WithExtractType("foo")),
			archiveName: "TestZip.gz",
			content:     compressGzip(t, []byte("foobar content")),
			gen:         createFile,
			expectError: true,
		},
		{
			name:          "get brotli extractor for file",
			cfg:           extract.NewConfig(),
			archiveName:   "TestBrotli.br",
			content:       compressBrotli(t, []byte("foobar content")),
			gen:           createFile,
			expectedFiles: []string{"TestBrotli"},
			expectError:   false,
		},
		{
			name:        "extract zip file inside a tar.gz archive with extract type set to tar.gz",
			cfg:         extract.NewConfig(extract.WithExtractType(extract.FileExtensionGZip)),
			archiveName: "example.json.zip.tar.gz",
			content: compressGzip(t, packTar(t, []archiveContent{
				{
					Content:    packZip(t, []archiveContent{{Name: "example.json", Content: []byte(`{"foo": "bar"}`)}}),
					Linktarget: "",
					Mode:       0644,
					Name:       "example.json.zip",
				},
			})),
			gen:           createFile,
			expectedFiles: []string{"example.json.zip"},
			expectError:   false,
		},
		{
			name:        "extract zip file inside a tar.gz archive with extract type set to zip, so that it fails",
			cfg:         extract.NewConfig(extract.WithExtractType(extract.FileExtensionZIP)),
			archiveName: "example.json.zip.tar.gz",
			content: compressGzip(t, packTar(t, []archiveContent{
				{
					Content:    packZip(t, []archiveContent{{Name: "example.json", Content: []byte(`{"foo": "bar"}`)}}),
					Linktarget: "",
					Mode:       0644,
					Name:       "example.json.zip",
				},
			})),
			gen:         createFile,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// perform actual tests
			archive := test.gen(filepath.Join(testDir, test.archiveName), test.content)
			err := extract.Unpack(
				context.Background(),
				archive,
				testDir,
				test.cfg,
			)
			defer func() {
				if closer, ok := archive.(io.Closer); ok {
					if closeErr := closer.Close(); closeErr != nil {
						t.Fatal(closeErr)
					}
				}
			}()

			if test.expectError && err == nil {
				t.Errorf("\nexpected error\ngot: %s", err)
			}

			// check for created files
			for _, file := range test.expectedFiles {
				_, err := os.Stat(filepath.Join(testDir, file))
				if err != nil {
					t.Errorf("\nexpected file: %s\ngot: %s", file, err)
				}
			}
		})
	}
}

// createFile creates a file with the given data and returns a reader for it.
func createFile(target string, data []byte) io.Reader {
	// Write the compressed data to the file
	if err := os.WriteFile(target, data, 0640); err != nil {
		panic(fmt.Errorf("error writing compressed data to file: %w", err))
	}

	// Open the file
	newFile, err := os.Open(target)
	if err != nil {
		panic(fmt.Errorf("error opening file: %w", err))
	}

	return newFile
}

func TestUnsupportedArchiveNames(t *testing.T) {
	// test cases
	cases := []struct {
		name        string
		createInput func(string) string
		windows     string
		other       string
	}{
		{
			name: "valid archive name (gzip)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "test.gz"}, string(filepath.Separator))
				createTestFile(t, fPath, string(compressGzip(t, []byte("foobar content"))))
				return fPath
			},
			windows: "test",
			other:   "test",
		},
		{
			name: "invalid reported 1 (..bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "..bz2"}, string(filepath.Separator))
				createTestFile(t, fPath, string(compressBzip2(t, []byte("foobar content"))))
				return fPath
			},
			windows: "goextract-decompressed-content",
			other:   "goextract-decompressed-content",
		},
		{
			name: "invalid reported 2 (test..bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "test..bz2"}, string(filepath.Separator))
				createTestFile(t, fPath, string(compressBzip2(t, []byte("foobar content"))))
				return fPath
			},
			windows: "test.",
			other:   "test.",
		},
		{
			name: "invalid reported 3 (test.bz2.)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "test.bz2."}, string(filepath.Separator))
				createTestFile(t, fPath, string(compressBzip2(t, []byte("foobar content"))))
				return fPath
			},
			windows: "test.bz2..decompressed",
			other:   "test.bz2..decompressed",
		},
		{
			name: "invalid reported 4 (....bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "....bz2"}, string(filepath.Separator))
				createTestFile(t, fPath, string(compressBzip2(t, []byte("foobar content"))))
				return fPath
			},
			windows: "goextract-decompressed-content",
			other:   "...",
		},
		{
			name: "invalid reported 5 (.. ..bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, ".. ..bz2"}, string(filepath.Separator))
				createTestFile(t, fPath, string(compressBzip2(t, []byte("foobar content"))))
				return fPath
			},
			windows: "goextract-decompressed-content",
			other:   ".. .",
		},
	}

	cfg := extract.NewConfig(extract.WithCreateDestination(true))

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {

			// prepare file
			tmpDir := t.TempDir()
			tmpFile := test.createInput(tmpDir)

			// run test
			archive, err := os.Open(tmpFile)
			if err != nil {
				t.Fatalf("error opening file: %s", err)
			}

			// perform actual tests
			ctx := context.Background()
			dstDir := filepath.Join(tmpDir, "out")
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				t.Fatalf("error creating directory: %s", err)
			}
			err = extract.Unpack(ctx, archive, dstDir, cfg)
			archive.Close()

			// check if error is expected
			if err != nil {
				t.Fatalf("\nexpected error: %v\ngot: %s\n", false, err)
			}

			// check for created files
			expectedFile := filepath.Join(tmpDir, "out", test.other)
			if runtime.GOOS == "windows" {
				expectedFile = filepath.Join(tmpDir, "out", test.windows)
			}
			if _, err := os.Stat(expectedFile); err != nil {
				t.Fatalf("\nexpected file: %s\ngot: %s\n", expectedFile, err)
			}
		})
	}
}

func TestWithCustomMode(t *testing.T) {
	umask := sniffUmask(t)

	tests := []struct {
		name        string
		data        []byte
		dst         string
		cfg         *extract.Config
		expected    map[string]fs.FileMode
		expectError bool
	}{
		{
			name: "dir with 0755 and file with 0644",
			data: compressGzip(t, packTar(t, []archiveContent{
				{
					Name: "sub/file",
					Mode: fs.FileMode(0644), // 420
				},
			})),
			cfg: extract.NewConfig(
				extract.WithCustomCreateDirMode(fs.FileMode(0755)), // 493
			),
			expected: map[string]fs.FileMode{
				"sub":      fs.FileMode(0755), // 493
				"sub/file": fs.FileMode(0644), // 420
			},
		},
		{
			name: "decompress with custom mode",
			data: compressGzip(t, []byte("foobar content")),
			dst:  "out", // specify decompressed file name
			cfg: extract.NewConfig(
				extract.WithCustomDecompressFileMode(fs.FileMode(0666)), // 438
			),
			expected: map[string]fs.FileMode{
				"out": fs.FileMode(0666), // 438
			},
		},
		{
			name:        "failing /bc of missing dir creation flag",
			data:        compressGzip(t, []byte("foobar content")),
			dst:         "foo/out", // specify decompressed file name in sub directory
			cfg:         extract.NewConfig(),
			expected:    nil, // should error, bc/ missing dir creation flag
			expectError: true,
		},
		{
			name: "dir with 0755 and file with 0777",
			data: compressGzip(t, []byte("foobar content")),
			dst:  "foo/out",
			cfg: extract.NewConfig(
				extract.WithCreateDestination(true),                     // create destination^
				extract.WithCustomCreateDirMode(fs.FileMode(0750)),      // 488
				extract.WithCustomDecompressFileMode(fs.FileMode(0777)), // 511
			),
			expected: map[string]fs.FileMode{
				"foo":     fs.FileMode(0750), // 488
				"foo/out": fs.FileMode(0777), // 511
			},
			expectError: false, // because its just a compressed byte slice without any directories specified and WithCreateDestination is not set
		},
		{
			name: "dir with 0777 and file with 0777",
			data: compressGzip(t, packTar(t, []archiveContent{
				{
					Name: "sub/file",
					Mode: fs.FileMode(0777), // 511
				},
			})),
			cfg: extract.NewConfig(
				extract.WithCustomCreateDirMode(fs.FileMode(0777)), // 511
			),
			expected: map[string]fs.FileMode{
				"sub":      fs.FileMode(0777), // 511
				"sub/file": fs.FileMode(0777), // 511
			},
		},
		{
			name: "file with 0000 permissions",
			data: compressGzip(t, packTar(t, []archiveContent{
				{
					Name: "file",
					Mode: fs.FileMode(0000), // 0
				},
				{
					Name: "dir/",
					Mode: fs.ModeDir, // 000 permission
				},
			})),
			cfg: extract.NewConfig(),
			expected: map[string]fs.FileMode{
				"file": fs.FileMode(0000), // 0
				"dir":  fs.FileMode(0000), // 0
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// prepare test
			buf := bytes.NewBuffer(test.data)
			ctx := context.Background()

			// create temp dir
			tmpDir := t.TempDir()
			dst := filepath.Join(tmpDir, test.dst)

			// run test
			err := extract.Unpack(ctx, buf, dst, test.cfg)
			if !test.expectError && (err != nil) {
				t.Fatal(err)
			}

			if test.expectError && (err == nil) {
				t.Fatal(err)
			}

			// check results
			for name, expectedMode := range test.expected {
				stat, err := os.Stat(filepath.Join(tmpDir, name))
				if err != nil {
					t.Fatal(err)
				}

				skip := false
				// adjust for windows
				if runtime.GOOS == "windows" {
					skip = stat.IsDir() // ignore directories to be checked on windows, reason is that the mode is not under control of the go code
					expectedMode = toWindowsFileMode(stat.IsDir(), expectedMode)
				} else {
					// adjust for umask
					expectedMode = expectedMode & ^umask
				}

				if !skip && stat.Mode().Perm() != expectedMode.Perm() {
					t.Fatalf("expected directory/file to have mode %s, but got: %s", expectedMode.Perm(), stat.Mode().Perm())
				}
			}
		})
	}
}

// sniffUmask is a helper function to get the umask
func sniffUmask(t *testing.T) fs.FileMode {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "file")

	// create 0777 file in temporary directory
	err := createTestFileWithPerm(tmpFile, "foobar content", 0777)
	if err != nil {
		t.Fatalf("error creating test file: %s", err)
	}

	// get stats
	stat, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("error getting file stats: %s", err)
	}

	// get umask
	umask := fs.FileMode(^stat.Mode().Perm() & 0777)

	// return the umask
	return umask
}

// toWindowsFileMode converts a fs.FileMode to a windows file mode
func toWindowsFileMode(isDir bool, mode fs.FileMode) fs.FileMode {

	// handle special case
	if isDir {
		return fs.FileMode(0777)
	}

	// check for write permission
	if mode&0200 != 0 {
		return fs.FileMode(0666)
	}

	// return the mode
	return fs.FileMode(0444)
}

// createTestFile is a helper function to generate test files
func createTestFileWithPerm(path string, content string, mode fs.FileMode) error {
	byteArray := []byte(content)
	return os.WriteFile(path, byteArray, mode)
}

func TestToWindowsFileMode(t *testing.T) {

	if runtime.GOOS != "windows" {
		t.Skip("skipping test on non-windows systems")
	}

	otherMasks := []int{00, 01, 02, 03, 04, 05, 06, 07}
	groupMasks := []int{00, 010, 020, 030, 040, 050, 060, 070}
	userMasks := []int{00, 0100, 0200, 0300, 0400, 0500, 0600, 0700}

	for _, dir := range []bool{true, false} {
		for _, o := range otherMasks {
			for _, g := range groupMasks {
				for _, u := range userMasks {

					// define test directory
					tmpDir := t.TempDir()
					fp := filepath.Join(tmpDir, "test")

					// define mode
					mode := fs.FileMode(u | g | o)

					// create test file or directory
					var err error
					if dir {
						err = os.MkdirAll(fp, mode)
					} else {
						err = createTestFileWithPerm(fp, "foobar content", mode)
					}
					if err != nil {
						t.Fatalf("error creating test file: %s", err)
					}

					// get stats
					stat, err := os.Stat(fp)
					if err != nil {
						t.Fatalf("error getting file stats: %s", err)
					}

					// calculate windows mode
					calculated := toWindowsFileMode(dir, mode)

					// check if the calculated mode is the same as the mode from the stat
					if stat.Mode().Perm() != calculated.Perm() {
						t.Errorf("toWindowsFileMode(%t, %s) calculated mode mode %s, but actual windows mode: %s", dir, mode, calculated.Perm(), stat.Mode().Perm())
					}

				}
			}
		}
	}
}
