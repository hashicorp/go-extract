// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/dsnet/compress/bzip2"
	"github.com/golang/snappy"
	"github.com/hashicorp/go-extract"
	"github.com/klauspost/compress/zstd"
	lz4 "github.com/pierrec/lz4/v4"
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
	if err := extract.Unpack(ctx, dst, src, cfg); err != nil {
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
		tm  = extract.NewTargetMemory() // create a new in-memory filesystem
		dst = ""                        // root of in-memory filesystem
		src = openFile("example.zip")   // source reader
		cfg = extract.NewConfig()       // custom config for extraction
	)

	// unpack
	if err := extract.UnpackTo(ctx, tm, dst, src, cfg); err != nil {
		// handle error
	}

	// read extracted file using fs package
	content, err := fs.ReadFile(tm, "example.txt")
	if err != nil {
		// handle error
	}
	fmt.Println(string(content))
	// Output:
	// example content
}

func ExampleNewTargetMemory() {
	var (
		ctx = context.Background()      // context for cancellation
		tm  = extract.NewTargetMemory() // create a new in-memory filesystem
		dst = ""                        // root of in-memory filesystem
		src = openFile("example.zip")   // source reader
		cfg = extract.NewConfig()       // custom config for extraction
	)

	// unpack
	if err := extract.UnpackTo(ctx, tm, dst, src, cfg); err != nil {
		// handle error
	}

	if err := fs.WalkDir(tm, ".", func(path string, d fs.DirEntry, err error) error {
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

func ExampleNewTargetDisk() {
	var (
		ctx = context.Background()    // context for cancellation
		td  = extract.NewTargetDisk() // local filesystem
		dst = createDirectory("out")  // create destination directory
		src = openFile("example.zip") // source reader
		cfg = extract.NewConfig()     // custom config for extraction
	)

	// unpack
	if err := extract.UnpackTo(ctx, td, dst, src, cfg); err != nil {
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

// Demonstrates how to check if a given file has a known archive extension.
func ExampleHasKnownArchiveExtension() {
	var (
		testFile = "example.zip" // source file
	)

	if extract.HasKnownArchiveExtension(testFile) {
		fmt.Println("test file is an archive")
	}
	// Output:
	// test file is an archive
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

	err := extract.Unpack(ctx, dst, src, cfg)
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

func TestUnpack(t *testing.T) {

	testCases := []struct {
		name        string
		archive     []byte
		cfg         *extract.Config
		expectError bool
	}{
		{
			name:    "single file",
			archive: packTar(t, []archiveContent{{Name: "test", Mode: 0640, Content: []byte("foobar content")}}),
		},
		{
			name:        "file with no name",
			archive:     packTar(t, []archiveContent{{Name: "", Mode: 0640, Content: []byte("foobar content")}}),
			expectError: true,
		},
		{
			name:        "symlink with no name",
			archive:     packTar(t, []archiveContent{{Name: "", Mode: fs.ModeSymlink | 0755, Linktarget: "foobar"}}),
			expectError: true,
		},
		{
			name:    "symlink with absolute path, but continue on error",
			archive: packTar(t, []archiveContent{{Name: "passwd", Mode: fs.ModeSymlink | 0755, Linktarget: "/etc/passwd"}}),
			cfg:     extract.NewConfig(extract.WithContinueOnError(true)),
		},
		{
			name:    "test rar",
			archive: packRar(t, []archiveContent{{Name: "test", Mode: 0640, Content: []byte("foobar content")}}),
			cfg: extract.NewConfig(
				extract.WithDenySymlinkExtraction(true),
				extract.WithContinueOnError(true),
				extract.WithCacheInMemory(true),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cfg == nil {
				tc.cfg = extract.NewConfig()
			}
			var (
				ctx = context.Background()
				dst = t.TempDir()
				src = asIoReader(t, tc.archive)
				cfg = tc.cfg
			)

			err := extract.Unpack(ctx, dst, src, cfg)
			if tc.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
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

			if err := extract.Unpack(ctx, dst, src, cfg); err != nil {
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
					cfg = extract.NewConfig(
						extract.WithCreateDestination(true),
						extract.WithContinueOnUnsupportedFiles(true),
					)
				)

				if err := extract.Unpack(ctx, dst, src, cfg); err != nil {
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
			err := extract.Unpack(ctx, dst, src, cfg)

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
				src = asIoReader(t, packZip(t, []archiveContent{
					{Name: tc, Content: []byte("hello world"), Mode: 0644},
				}))
			)
			if err := extract.Unpack(ctx, dst, src, extract.NewConfig()); err == nil {
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
			Mode:       fs.ModeSymlink | 0755,
			Linktarget: "../test",
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
			name: "unpack with overwrite enabled (files)",
			testArchive: []archiveContent{
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
			},
			cfg:         extract.NewConfig(extract.WithOverwrite(true)),
			expectError: false,
		},
		{
			name: "unpack with overwrite enabled (symlink)",
			testArchive: []archiveContent{
				{Name: "test", Content: []byte("hello world"), Mode: 0644},
				{Name: "link", Mode: fs.ModeSymlink | 0755, Linktarget: "test"},
				{Name: "link", Mode: fs.ModeSymlink | 0755, Linktarget: "test"},
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
			cfg:         extract.NewConfig(extract.WithInsecureTraverseSymlinks(true)),
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
				err := extract.Unpack(ctx, dst, src, cfg)
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
			err := extract.Unpack(ctx, dst, tc.src, cfg)
			if tc.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestUnpack is a test function
func TestUnpackToMemory(t *testing.T) {

	c := []archiveContent{
		{Name: "test", Content: []byte(strings.Repeat("A", 1024)), Mode: 0644},
		{Name: "dir", Mode: fs.ModeDir | 0755},
		{Name: "dir/link", Linktarget: "../test", Mode: fs.ModeSymlink | 0755},
	}
	randomBytes := []byte(strings.Repeat("A", 1024))

	tests := []struct {
		name        string
		src         io.Reader
		expectError bool
	}{
		{
			name: "Unzip",
			src:  asIoReader(t, packZip(t, c)),
		},
		{
			name: "untar",
			src:  asIoReader(t, packTar(t, c)),
		},
		{
			name: "gunzip",
			src:  asIoReader(t, compressGzip(t, randomBytes)),
		},
		{
			name:        "rubbish",
			src:         asIoReader(t, []byte("rubbish")),
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				ctx = context.Background()
				tm  = extract.NewTargetMemory()
				dst = ""
				cfg = extract.NewConfig()
			)
			err := extract.UnpackTo(ctx, tm, dst, test.src, cfg)
			if test.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTelemetryHook(t *testing.T) {

	oneFile := []archiveContent{
		{Name: "test", Content: []byte(strings.Repeat("A", 1024)), Mode: 0644},
	}

	fiveFiles := []archiveContent{
		{Name: "test0", Content: []byte(strings.Repeat("A", 1024)), Mode: 0644},
		{Name: "test1", Content: []byte(strings.Repeat("A", 1024)), Mode: 0644},
		{Name: "test2", Content: []byte(strings.Repeat("A", 1024)), Mode: 0644},
		{Name: "test3", Content: []byte(strings.Repeat("A", 1024)), Mode: 0644},
		{Name: "test4", Content: []byte(strings.Repeat("A", 1024)), Mode: 0644},
	}

	tests := []struct {
		name                  string
		archive               []byte
		cfgOps                []extract.ConfigOption
		expectedTelemetryData *extract.TelemetryData
		expectError           bool
		dst                   string
	}{
		{
			name:    "normal gzip with file",
			archive: compressGzip(t, []byte(strings.Repeat("A", 1024))),
			cfgOps: []extract.ConfigOption{
				extract.WithMaxExtractionSize(1024),
				extract.WithMaxFiles(1),
			},
			expectedTelemetryData: &extract.TelemetryData{
				ExtractedFiles: 1,
				ExtractionSize: 1024,
				ExtractedType:  "gz",
			},
		},
		{
			name:    "normal gzip with file and decompression target-name in sub-dir failing",
			archive: compressGzip(t, []byte(strings.Repeat("A", 1024))),
			dst:     "sub/target", // important: the gzip decompression has a filename as dst
			cfgOps: []extract.ConfigOption{
				extract.WithMaxExtractionSize(1024),
				extract.WithMaxFiles(1),
			},
			expectedTelemetryData: &extract.TelemetryData{
				ExtractionErrors: 1,
				ExtractedType:    "gz",
			},
			expectError: true,
		},
		{
			name:    "normal gzip with file, and decompression target-name in sub-dir with sub-dir-creation",
			archive: compressGzip(t, []byte(strings.Repeat("A", 1024))),
			dst:     "sub/target", // important: the gzip decompression has a filename das dst
			cfgOps:  []extract.ConfigOption{extract.WithCreateDestination(true)},
			expectedTelemetryData: &extract.TelemetryData{
				ExtractedFiles: 1,
				ExtractionSize: 1024,
				ExtractedType:  "gz",
			},
		},
		{
			name:    "normal tar with file",
			archive: packTar(t, oneFile),
			expectedTelemetryData: &extract.TelemetryData{
				ExtractedFiles: 1,
				ExtractionSize: 1024,
				ExtractedType:  "tar",
			},
		},
		{
			name:    "normal tar.gz with 5 files",
			archive: packTar(t, fiveFiles),
			expectedTelemetryData: &extract.TelemetryData{
				ExtractedFiles: 5,
				ExtractionSize: 1024 * 5,
				ExtractedType:  "tar",
			},
		},
		{
			name:    "normal tar.gz with file with max files limit",
			archive: packTar(t, fiveFiles),
			cfgOps:  []extract.ConfigOption{extract.WithMaxFiles(4)},
			expectedTelemetryData: &extract.TelemetryData{
				ExtractedFiles:      4,
				ExtractionErrors:    1,
				ExtractionSize:      1024 * 4,
				ExtractedType:       "tar",
				LastExtractionError: fmt.Errorf("max objects check failed: %w", extract.ErrMaxFilesExceeded),
			},
			expectError: true,
		},
		{
			name:    "normal tar.gz with file failing bc/ of missing sub directory",
			archive: packTar(t, fiveFiles),
			dst:     "sub",
			cfgOps:  []extract.ConfigOption{extract.WithContinueOnError(true)},
			expectedTelemetryData: &extract.TelemetryData{
				ExtractionErrors: 5,
				ExtractedType:    "tar",
			},
		},
		{
			name:    "normal zip file",
			archive: packZip(t, oneFile),
			expectedTelemetryData: &extract.TelemetryData{
				ExtractedFiles: 1,
				ExtractionSize: 1024,
				ExtractedType:  "zip",
			},
		},
		{
			name:    "normal zip file extraction size exceeded",
			archive: packZip(t, oneFile),
			cfgOps:  []extract.ConfigOption{extract.WithMaxExtractionSize(512)},
			expectedTelemetryData: &extract.TelemetryData{
				ExtractionErrors:    1,
				ExtractedType:       "zip",
				LastExtractionError: fmt.Errorf("max extraction size exceeded: %w", extract.ErrMaxExtractionSizeExceeded),
			},
			expectError: true,
		},
	}

	tdEquals := func(td, other *extract.TelemetryData) bool {
		if td == nil && other == nil {
			return true
		}
		if td == nil || other == nil {
			return false
		}
		return td.ExtractedDirs == other.ExtractedDirs &&
			td.ExtractionErrors == other.ExtractionErrors &&
			td.ExtractedFiles == other.ExtractedFiles &&
			td.ExtractionSize == other.ExtractionSize &&
			td.ExtractedSymlinks == other.ExtractedSymlinks &&
			td.ExtractedType == other.ExtractedType &&
			td.PatternMismatches == other.PatternMismatches &&
			td.UnsupportedFiles == other.UnsupportedFiles &&
			td.LastUnsupportedFile == other.LastUnsupportedFile
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var (
				ctx     = context.Background()
				testDir = t.TempDir()
				src     = asIoReader(t, tc.archive)
				dst     = filepath.Join(testDir, tc.dst)
				td      *extract.TelemetryData
				hook    = func(ctx context.Context, d *extract.TelemetryData) {
					td = d
				}
				cfg = extract.NewConfig(append(tc.cfgOps, extract.WithTelemetryHook(hook))...)
			)
			if tc.expectedTelemetryData.InputSize == 0 {
				tc.expectedTelemetryData.InputSize = int64(len(tc.archive))
			}
			err := extract.Unpack(ctx, dst, src, cfg)
			if tc.expectError != (err != nil) {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, tc.expectError, err)
			}
			t.Logf("expected telemetry data: %s", tc.expectedTelemetryData.String())
			t.Logf("collected telemetry data: %s", td.String())
			if !tdEquals(tc.expectedTelemetryData, td) {
				t.Errorf("test case %d failed: %s\nexpected: %v\ngot: %v", i, tc.name, tc.expectedTelemetryData, td)
			}
		})
	}
}

func TestUnpackWithTypes(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *extract.Config
		src         io.Reader
		expectError bool
	}{
		{
			name: "Fix extraction to gunzip",
			cfg:  extract.NewConfig(extract.WithExtractType("tgz")),
			src:  createFileReader(t, "test*.gz", compressGzip(t, []byte("foobar content")))},
		{
			name:        "Non valid extraction type",
			cfg:         extract.NewConfig(extract.WithExtractType("foo")),
			src:         createFileReader(t, "test*.gz", compressGzip(t, []byte("foobar content"))),
			expectError: true,
		},
		{
			name:        "get brotli extractor for file",
			src:         createFileReader(t, "test*.br", compressBrotli(t, []byte("foobar content"))),
			expectError: false,
		},
		{
			name:        "extract zip file inside a tar.gz archive with extract type set to tar.gz",
			cfg:         extract.NewConfig(extract.WithExtractType("gz")),
			src:         createFileReader(t, "test*.tar.gz", compressGzip(t, packTar(t, []archiveContent{{Name: "test", Content: []byte("foobar content")}}))),
			expectError: false,
		},
		{
			name:        "extract zip file inside a tar.gz archive with extract type set to zip, so that it fails",
			cfg:         extract.NewConfig(extract.WithExtractType("zip")),
			src:         createFileReader(t, "example.json.zip*.tar.gz", compressGzip(t, packTar(t, []archiveContent{{Name: "example.json.zip", Content: packZip(t, []archiveContent{{Name: "example.json", Content: []byte(`{"foo": "bar"}`)}})}}))),
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.cfg == nil {
				test.cfg = extract.NewConfig()
			}
			var (
				ctx     = context.Background()
				testDir = t.TempDir()
				dst     = testDir
				cfg     = test.cfg
			)
			err := extract.Unpack(ctx, dst, test.src, cfg)
			if test.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUnsupportedArchiveNames(t *testing.T) {
	// test testCases
	testCases := []struct {
		name            string
		fileName        string
		expectOnWindows string
		expectOnOther   string
	}{
		{
			name:            "valid archive name (bz2)",
			fileName:        "test.bz2",
			expectOnWindows: "test",
			expectOnOther:   "test",
		},
		{
			name:            "invalid reported 1 (..bz2)",
			fileName:        "..bz2",
			expectOnWindows: "goextract-decompressed-content",
			expectOnOther:   "goextract-decompressed-content",
		},
		{
			name:            "invalid reported 2 (test..bz2)",
			fileName:        "test..bz2",
			expectOnWindows: "test.",
			expectOnOther:   "test.",
		},
		{
			name:            "invalid reported 3 (test.bz2.)",
			fileName:        "test.bz2.",
			expectOnWindows: "test.bz2..decompressed",
			expectOnOther:   "test.bz2..decompressed",
		},
		{
			name:            "invalid reported 4 (....bz2)",
			fileName:        "....bz2",
			expectOnWindows: "goextract-decompressed-content",
			expectOnOther:   "...",
		},
		{
			name:            "invalid reported 5 (.. ..bz2)",
			fileName:        ".. ..bz2",
			expectOnWindows: "goextract-decompressed-content",
			expectOnOther:   ".. .",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ctx                   = context.Background()
				tmpDir                = t.TempDir()
				tmpFile               = filepath.Join(tmpDir, tc.fileName)
				fileContent           = []byte("foobar content")
				compressedFileContent = compressBzip2(t, fileContent)
				expectedFile          = filepath.Join(tmpDir, tc.expectOnOther)
			)
			if runtime.GOOS == "windows" {
				expectedFile = filepath.Join(tmpDir, tc.expectOnWindows)
			}
			if err := os.WriteFile(tmpFile, compressedFileContent, 0644); err != nil {
				t.Fatalf("error writing file: %s", err)
			}
			src, err := os.Open(tmpFile)
			if err != nil {
				t.Fatalf("error opening file: %s", err)
			}
			defer src.Close()
			if err := extract.Unpack(ctx, tmpDir, src, extract.NewConfig()); err != nil {
				t.Fatalf("error unpacking file: %s", err)
			}
			if _, err := os.Stat(expectedFile); err != nil {
				t.Fatalf("\nexpected file: %s\ngot: %s\n", expectedFile, err)
			}
		})
	}
}

func TestHasKnownArchiveExtension(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected bool
	}{
		{
			name:     "valid archive name (bz2)",
			fileName: "test.bz2",
			expected: true,
		},
		{
			name:     "valid archive name (gz)",
			fileName: "test.gz",
			expected: true,
		},
		{
			name:     "valid archive name (tar.gz)",
			fileName: "test.tar.gz",
			expected: true,
		},
		{
			name:     "valid archive name (tar.bz2)",
			fileName: "test.tar.bz2",
			expected: true,
		},
		{
			name:     "valid archive name (zip)",
			fileName: "test.zip",
			expected: true,
		},
		{
			name:     "valid archive name (tgz)",
			fileName: "test.tgz",
			expected: true,
		},
		{
			name:     "valid archive name (tar.xz)",
			fileName: "test.tar.xz",
			expected: true,
		},
		{
			name:     "valid archive name (tar.lz4)",
			fileName: "test.tar.lz4",
			expected: true,
		},
		{
			name:     "valid archive name (tar.zst)",
			fileName: "test.tar.zst",
			expected: true,
		},
		{
			name:     "valid archive name (tar.sz)",
			fileName: "test.tar.sz",
			expected: true,
		},
		{
			name:     "valid archive name (tar)",
			fileName: "test.tar",
			expected: true,
		},
		{
			name:     "valid archive name (tar.lz4)",
			fileName: "test.tar.lz4",
			expected: true,
		},
		{
			name:     "invalid archive name (tar.txt)",
			fileName: "test.tar.txt",
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if extract.HasKnownArchiveExtension(tc.fileName) != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, !tc.expected)
			}
		})
	}
}

func abs(v int64) int64 {
	return int64(math.Abs(float64(v)))
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
	AccessTime time.Time
	ModTime    time.Time
	Uid        int
	Gid        int
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
		header.Uid = c.Uid
		header.Gid = c.Gid
		header.AccessTime = c.AccessTime
		header.ModTime = c.ModTime
		if tFlag == tar.TypeXGlobalHeader {
			header.Mode = 0
			header.Size = 0
			header.Uid = 0
			header.Gid = 0
			header.AccessTime = time.Time{}
			header.ModTime = time.Time{}
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
		h.Modified = c.ModTime
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

// pack7z2 creates always the same a 7z archive with following files:
// -rw-r--r--  1 503  20    27B  6 Dez 14:12 test
// drwxr-xr-x  3 503  20    96B  6 Dez 14:12 sub/
// -rw-r--r--  1 503  20    27B  6 Dez 14:12 sub/test
// lrwxr-xr-x  1 503  20     8B  6 Dez 14:12 link@ -> sub/test
var contents7z2 = []archiveContent{
	{Name: "test", Content: []byte("hello world"), Mode: 0644, AccessTime: time.Date(2024, 12, 6, 14, 12, 0, 0, time.Local), ModTime: time.Date(2024, 12, 6, 13, 12, 42, 315443500, time.UTC), Uid: 503, Gid: 20},
	{Name: "sub", Mode: fs.ModeDir | 0755, AccessTime: time.Date(2024, 12, 6, 14, 12, 0, 0, time.Local), ModTime: time.Date(2024, 12, 6, 13, 12, 49, 378600200, time.UTC), Uid: 503, Gid: 20},
	{Name: "sub/test", Content: []byte("hello world"), Mode: 0644, AccessTime: time.Date(2024, 12, 6, 14, 12, 0, 0, time.Local), ModTime: time.Date(2024, 12, 6, 13, 12, 49, 378790200, time.UTC), Uid: 503, Gid: 20},
	{Name: "link", Linktarget: "sub/test", Mode: fs.ModeSymlink | 0755, AccessTime: time.Date(2024, 12, 6, 14, 12, 0, 0, time.Local), ModTime: time.Date(2024, 12, 6, 13, 12, 54, 532031200, time.UTC), Uid: 503, Gid: 20},
}

func pack7z2(t *testing.T, _ []archiveContent) []byte {
	t.Helper()
	b, err := hex.DecodeString("377abcaf271c00042d5fc057b50000000000000022000000000000004e8d3aa1e0003d00285d00399d486415d3bb7a709d8c05b9a4f8a601c485ca32a1ba56fbed0277df127ac8b5849a02ef89b000000000813307ae0fd100d43ca090a0775ec540189123d516c0a4234b6046777137a236d0c100afd4540a63bac5dbcdd5f4954e1321f89bc2fee32eda1ffebe24d8ec7f5495f31cb107f418f1a438bedfa190f8d5e9bd34f41831a3e85fb8590ee2d3eb6854856ce91c64623e7b1bec5c6bf403f9b195d06eb0810540f173e9abd2005e6a00001706300109808500070b01000123030101055d001000000c80ae0a01d53cb2d70000")
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

// packRar2 creates always the same a rar archive with following files:
// -rw-r--r--  1 503  20    27B  6 Dez 14:07 test
// drwxr-xr-x  3 503  20    96B  6 Dez 14:08 sub/
// -rw-r--r--  1 503  20    27B  6 Dez 14:08 sub/test
var contentsRar2 = []archiveContent{
	{Name: "test", Content: []byte("hello world"), Mode: 0644, AccessTime: time.Date(2024, 12, 6, 14, 7, 0, 0, time.Local), ModTime: time.Date(2024, 12, 6, 14, 8, 0, 0, time.Local), Uid: 503, Gid: 20},
	{Name: "sub", Mode: fs.ModeDir | 0755, AccessTime: time.Date(2024, 12, 6, 14, 8, 0, 0, time.Local), ModTime: time.Date(2024, 12, 6, 14, 7, 8, 0, time.Local), Uid: 503, Gid: 20},
	{Name: "sub/test", Content: []byte("hello world"), Mode: 0644, AccessTime: time.Date(2024, 12, 6, 14, 8, 0, 0, time.Local), ModTime: time.Date(2024, 12, 6, 14, 8, 0, 0, time.Local), Uid: 503, Gid: 20},
}

func packRar2(t *testing.T, _ []archiveContent) []byte {
	t.Helper()
	b, err := hex.DecodeString("526172211a0701003392b5e50a010506000501018080003afe2e322202030b9b00049b00a48302032d6c9680000104746573740a03132ff752678a911e136861736869207361797320686920746f2074686520776f726c640a7db74f802602030b9b00049b00a48302032d6c96800001087375622f746573740a031334f752672333f02b6861736869207361797320686920746f2074686520776f726c640a5311ba9e1b02030b000100ed8301800001037375620a031334f752673549ed2b1d77565103050400")
	if err != nil {
		t.Fatalf("error decoding rar data: %v", err)
	}
	return b
}

var (
	testDataUid, testDataGid          = 1337, 42
	testDataRootUid, testDataWheelGid = 0, 0
	baseTime                          = time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
)

var testCases = []struct {
	name                  string
	contents              []archiveContent
	packer                func(*testing.T, []archiveContent) []byte
	doesNotSupportModTime bool
	doesNotSupportOwner   bool
	expectError           bool
}{
	{
		name: "tar",
		contents: []archiveContent{
			{Name: "test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: testDataUid, Gid: testDataGid},
			{Name: "sub", Mode: fs.ModeDir | 0777, AccessTime: baseTime, ModTime: baseTime, Uid: testDataUid, Gid: testDataGid},
			{Name: "sub/test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: testDataUid, Gid: testDataGid},
			{Name: "link", Mode: fs.ModeSymlink | 0777, Linktarget: "sub/test", AccessTime: baseTime, ModTime: baseTime, Uid: testDataUid, Gid: testDataGid},
		},
		packer: packTar,
	},
	{
		name: "root-tar",
		contents: []archiveContent{
			{Name: "test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: testDataRootUid, Gid: testDataWheelGid},
			{Name: "sub", Mode: fs.ModeDir | 0777, AccessTime: baseTime, ModTime: baseTime, Uid: testDataRootUid, Gid: testDataWheelGid},
			{Name: "sub/test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: testDataRootUid, Gid: testDataWheelGid},
			{Name: "link", Mode: fs.ModeSymlink | 0777, Linktarget: "sub/test", AccessTime: baseTime, ModTime: baseTime, Uid: testDataRootUid, Gid: testDataWheelGid},
		},
		packer: packTar,
	},
	{
		name: "zip",
		contents: []archiveContent{
			{Name: "test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: os.Getuid(), Gid: os.Getgid()},
			{Name: "sub", Mode: fs.ModeDir | 0777, AccessTime: baseTime, ModTime: baseTime, Uid: os.Getuid(), Gid: os.Getgid()},
			{Name: "sub/test", Content: []byte("hello world"), Mode: 0644, AccessTime: baseTime, ModTime: baseTime, Uid: os.Getuid(), Gid: os.Getgid()},
			{Name: "link", Mode: fs.ModeSymlink | 0777, Linktarget: "sub/test", AccessTime: baseTime, ModTime: baseTime, Uid: os.Getuid(), Gid: os.Getgid()},
		},
		doesNotSupportOwner: true,
		packer:              packZip,
	},
	{
		name:                  "rar",
		contents:              contentsRar2,
		doesNotSupportOwner:   true,
		doesNotSupportModTime: true,
		packer:                packRar2,
	},
	{
		name:                "7z",
		contents:            contents7z2,
		doesNotSupportOwner: true,
		packer:              pack7z2,
	},
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
		if _, err := w.Write(b); err != nil {
			t.Logf("error writing data to pipe: %v", err)
			t.Fail()
		}
	}()
	return r
}

// sniffUmask is a helper function to get the umask
func sniffUmask(t *testing.T) fs.FileMode {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "file")

	// create 0777 file in temporary directory
	err := os.WriteFile(tmpFile, []byte("foobar content"), 0777)
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
