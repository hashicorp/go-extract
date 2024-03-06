package extractor

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

// TestTarUnpack implements test cases
func TestTarUnpackNew(t *testing.T) {

	// generate cancled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	cases := []struct {
		name        string
		content     []tarContent
		opts        []config.ConfigOption
		expectError bool
		ctx         context.Context
	}{
		{
			name:        "unpack normal tar",
			content:     []tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}},
			expectError: false,
		},
		{
			name:        "unpack normal tar, but pattern mismatch",
			content:     []tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}},
			opts:        []config.ConfigOption{config.WithPatterns("*foo")},
			expectError: false,
		},
		{
			name:        "unpack normal tar, but context canceled",
			content:     []tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}},
			ctx:         canceledCtx,
			expectError: true,
		},
		{
			name: "unpack normal tar with 5 files",
			content: []tarContent{
				{Content: []byte("foobar content"), Name: "test1", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test2", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test3", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test4", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test5", Mode: 0640, Filetype: tar.TypeReg},
			},
			expectError: false,
		},
		{
			name: "unpack normal tar with 5 files, but file limit",
			content: []tarContent{
				{Content: []byte("foobar content"), Name: "test1", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test2", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test3", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test4", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test5", Mode: 0640, Filetype: tar.TypeReg},
			},
			opts:        []config.ConfigOption{config.WithMaxFiles(4)},
			expectError: true,
		},
		{
			name:        "unpack normal tar, but extraction size exceeded",
			content:     []tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}},
			opts:        []config.ConfigOption{config.WithMaxExtractionSize(1)},
			expectError: true,
		},
		{
			name:        "unpack malicious tar, with traversal",
			content:     []tarContent{{Content: []byte("foobar content"), Name: "../test", Mode: 0640, Filetype: tar.TypeReg}},
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink",
			content:     []tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}},
			expectError: false,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     []tarContent{{Name: "../test", Filetype: tar.TypeDir}},
			expectError: true,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     []tarContent{{Name: "../test", Filetype: tar.TypeDir}},
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name:        "unpack normal tar with traversal symlink",
			content:     []tarContent{{Name: "foo", Linktarget: "../bar", Filetype: tar.TypeLink}},
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink, but symlinks are denied",
			content:     []tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}},
			opts:        []config.ConfigOption{config.WithDenySymlinkExtraction(true)},
			expectError: true,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on error",
			content: []tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}},
			opts: []config.ConfigOption{
				config.WithDenySymlinkExtraction(true),
				config.WithContinueOnError(true),
			},
			expectError: false,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on unsupported files",
			content: []tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}},
			opts: []config.ConfigOption{
				config.WithDenySymlinkExtraction(true),
				config.WithContinueOnUnsupportedFiles(true),
			},
			expectError: false,
		},
		{
			name:        "unpack normal tar with absolute path in symlink",
			content:     []tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "/absolute-target"}},
			expectError: true,
		},
		{
			name:        "malicious tar with symlink name path traversal",
			content:     []tarContent{{Name: "../testLink", Filetype: tar.TypeSymlink, Linktarget: "target"}},
			expectError: true,
		},
		{
			name:        "malicious tar with .. as filename",
			content:     []tarContent{{Content: []byte("foobar content"), Name: "..", Filetype: tar.TypeReg}},
			expectError: true,
		},
		{
			name:        "malicious tar with . as filename",
			content:     []tarContent{{Content: []byte("foobar content"), Name: ".", Filetype: tar.TypeReg}},
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype",
			content:     []tarContent{{Name: "fifo", Filetype: tar.TypeFifo}},
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype, but continue on error",
			content:     []tarContent{{Name: "fifo", Filetype: tar.TypeFifo}},
			opts:        []config.ConfigOption{config.WithContinueOnUnsupportedFiles(true)},
			expectError: false,
		},
		{
			name: "malicious tar with zip slip attack",
			content: []tarContent{
				{Name: "sub/to-parent", Filetype: tar.TypeSymlink, Linktarget: "../"},
				{Name: "sub/to-parent/one-above", Filetype: tar.TypeSymlink, Linktarget: "../"},
			},
			expectError: true,
		},
		{
			name:        "tar with legit git pax_global_header",
			content:     []tarContent{{Content: []byte(""), Name: "pax_global_header", Filetype: tar.TypeXGlobalHeader}},
			expectError: false,
		},
		{
			name: "absolute path in filename (windows)",
			content: []tarContent{
				{Content: []byte("foobar content"), Name: "c:\\absolute-path", Mode: 0640, Filetype: tar.TypeReg},
			},
			expectError: false,
		},
		{
			name: "absolute path in filename",
			content: []tarContent{
				{Content: []byte("foobar content"), Name: "/absolute-path", Mode: 0640, Filetype: tar.TypeReg},
			},
			expectError: false,
		},
		{
			name: "extract a directory",
			content: []tarContent{
				{Name: "test", Filetype: tar.TypeDir},
			},
			expectError: false,
		},
		{
			name: "extract a file with traversal, but continue on error",
			content: []tarContent{
				{Content: []byte("foobar content"), Name: "../test", Mode: 0640, Filetype: tar.TypeReg},
			},
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "extract a symlink with traversal, but continue on error",
			content: []tarContent{
				{Name: "foo", Linktarget: "../bar", Filetype: tar.TypeSymlink},
			},
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "tar with hard link, with error, but continue on error",
			content: []tarContent{
				{Name: "testLink", Filetype: tar.TypeLink, Linktarget: "testTarget"},
			},
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			testDir := t.TempDir()

			if tc.ctx == nil {
				tc.ctx = context.Background()
			}
			ctx := tc.ctx

			// perform actual tests
			input := createTarWithContent(filepath.Join(testDir, "test.tar"), tc.content)
			want := tc.expectError
			err := UnpackTar(ctx, input, testDir, config.NewConfig(tc.opts...))
			defer input.(io.Closer).Close()
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}
}

// tarContent is a struct to store the content of a tar file
type tarContent struct {
	Content    []byte
	Linktarget string
	Mode       os.FileMode
	Name       string
	Filetype   byte
}

// createTarWithContent creates a tar file with the given content
func createTarWithContent(target string, content []tarContent) io.Reader {

	// create tar file
	file, tw := createTar(target)
	defer file.Close()

	// write content
	for _, c := range content {

		// create header
		hdr := &tar.Header{
			Name:     c.Name,
			Mode:     int64(c.Mode),
			Size:     int64(len(c.Content)),
			Linkname: c.Linktarget,
			Typeflag: c.Filetype,
		}

		// write header
		if err := tw.WriteHeader(hdr); err != nil {
			panic(err)
		}

		// write data
		if _, err := tw.Write(c.Content); err != nil {
			panic(err)
		}
	}

	// close tar writer
	if err := tw.Close(); err != nil {
		panic(err)
	}

	// return reader
	file, err := os.Open(target)
	if err != nil {
		panic(err)
	}
	return file
}

// createTar is a helper function to generate test content
func createTar(filePath string) (*os.File, *tar.Writer) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	return f, tar.NewWriter(f)
}
