package extractor

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestTarUnpack implements test cases
func TestTarUnpackNew(t *testing.T) {

	// generate cancled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	cases := []struct {
		name        string
		content     []byte
		opts        []config.ConfigOption
		expectError bool
		ctx         context.Context
	}{
		{
			name:        "unpack normal tar",
			content:     packTarWithContent([]tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			expectError: false,
		},
		{
			name:        "unpack normal tar, but pattern mismatch",
			content:     packTarWithContent([]tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			opts:        []config.ConfigOption{config.WithPatterns("*foo")},
			expectError: false,
		},
		{
			name:        "unpack normal tar, but context canceled",
			content:     packTarWithContent([]tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			ctx:         canceledCtx,
			expectError: true,
		},
		{
			name: "unpack normal tar with 5 files",
			content: packTarWithContent([]tarContent{
				{Content: []byte("foobar content"), Name: "test1", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test2", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test3", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test4", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test5", Mode: 0640, Filetype: tar.TypeReg},
			}),
			expectError: false,
		},
		{
			name: "unpack normal tar with 5 files, but file limit",
			content: packTarWithContent([]tarContent{
				{Content: []byte("foobar content"), Name: "test1", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test2", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test3", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test4", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test5", Mode: 0640, Filetype: tar.TypeReg},
			}),
			opts:        []config.ConfigOption{config.WithMaxFiles(4)},
			expectError: true,
		},
		{
			name:        "unpack normal tar, but extraction size exceeded",
			content:     packTarWithContent([]tarContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			opts:        []config.ConfigOption{config.WithMaxExtractionSize(1)},
			expectError: true,
		},
		{
			name:        "unpack malicious tar, with traversal",
			content:     packTarWithContent([]tarContent{{Content: []byte("foobar content"), Name: "../test", Mode: 0640, Filetype: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink",
			content:     packTarWithContent([]tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			expectError: false,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     packTarWithContent([]tarContent{{Name: "../test", Filetype: tar.TypeDir}}),
			expectError: true,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     packTarWithContent([]tarContent{{Name: "../test", Filetype: tar.TypeDir}}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name:        "unpack normal tar with traversal symlink",
			content:     packTarWithContent([]tarContent{{Name: "foo", Linktarget: "../bar", Filetype: tar.TypeLink}}),
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink, but symlinks are denied",
			content:     packTarWithContent([]tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			opts:        []config.ConfigOption{config.WithDenySymlinkExtraction(true)},
			expectError: true,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on error",
			content: packTarWithContent([]tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			opts: []config.ConfigOption{
				config.WithDenySymlinkExtraction(true),
				config.WithContinueOnError(true),
			},
			expectError: false,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on unsupported files",
			content: packTarWithContent([]tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			opts: []config.ConfigOption{
				config.WithDenySymlinkExtraction(true),
				config.WithContinueOnUnsupportedFiles(true),
			},
			expectError: false,
		},
		{
			name:        "unpack normal tar with absolute path in symlink",
			content:     packTarWithContent([]tarContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "/absolute-target"}}),
			expectError: runtime.GOOS != "windows",
		},
		{
			name:        "malicious tar with symlink name path traversal",
			content:     packTarWithContent([]tarContent{{Name: "../testLink", Filetype: tar.TypeSymlink, Linktarget: "target"}}),
			expectError: true,
		},
		{
			name:        "malicious tar with .. as filename",
			content:     packTarWithContent([]tarContent{{Content: []byte("foobar content"), Name: "..", Filetype: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "malicious tar with . as filename",
			content:     packTarWithContent([]tarContent{{Content: []byte("foobar content"), Name: ".", Filetype: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype",
			content:     packTarWithContent([]tarContent{{Name: "fifo", Filetype: tar.TypeFifo}}),
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype, but continue on error",
			content:     packTarWithContent([]tarContent{{Name: "fifo", Filetype: tar.TypeFifo}}),
			opts:        []config.ConfigOption{config.WithContinueOnUnsupportedFiles(true)},
			expectError: false,
		},
		{
			name: "malicious tar with zip slip attack",
			content: packTarWithContent([]tarContent{
				{Name: "sub/to-parent", Filetype: tar.TypeSymlink, Linktarget: "../"},
				{Name: "sub/to-parent/one-above", Filetype: tar.TypeSymlink, Linktarget: "../"},
			}),
			expectError: true,
		},
		{
			name:        "tar with legit git pax_global_header",
			content:     packTarWithContent([]tarContent{{Content: []byte(""), Name: "pax_global_header", Filetype: tar.TypeXGlobalHeader}}),
			expectError: false,
		},
		{
			name: "absolute path in filename (windows), fails bc of ':' in path",
			content: packTarWithContent([]tarContent{
				{Content: []byte("foobar content"), Name: "s:\\absolute-path", Mode: 0640, Filetype: tar.TypeReg},
			}),
			expectError: runtime.GOOS == "windows",
		},
		{
			name: "absolute path in filename",
			content: packTarWithContent([]tarContent{
				{Content: []byte("foobar content"), Name: "/absolute-path", Mode: 0640, Filetype: tar.TypeReg},
			}),
			expectError: false,
		},
		{
			name: "extract a directory",
			content: packTarWithContent([]tarContent{
				{Name: "test", Filetype: tar.TypeDir},
			}),
			expectError: false,
		},
		{
			name: "extract a file with traversal, but continue on error",
			content: packTarWithContent([]tarContent{
				{Content: []byte("foobar content"), Name: "../test", Mode: 0640, Filetype: tar.TypeReg},
			}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "extract a symlink with traversal, but continue on error",
			content: packTarWithContent([]tarContent{
				{Name: "foo", Linktarget: "../bar", Filetype: tar.TypeSymlink},
			}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "tar with hard link, with error, but continue on error",
			content: packTarWithContent([]tarContent{
				{Name: "testLink", Filetype: tar.TypeLink, Linktarget: "testTarget"},
			}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// Create a new target
			testingTarget := target.NewOS()

			// create testing directory
			testDir := t.TempDir()

			if tc.ctx == nil {
				tc.ctx = context.Background()
			}
			ctx := tc.ctx

			// perform actual tests
			input := newTestFile(filepath.Join(testDir, "test.tar"), tc.content)
			want := tc.expectError
			err := UnpackTar(ctx, testingTarget, testDir, input, config.NewConfig(tc.opts...))
			defer input.(io.Closer).Close()
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}
}

func TestIsTar(t *testing.T) {

	tc := []struct {
		Name    string
		Content []byte
	}{
		{
			Name:    "Tar header 'magicGNU/versionGNU'",
			Content: []byte("ustar\x00tar\x00"),
		},
		{
			Name:    "Tar header 'magicUSTAR/versionUSTAR'",
			Content: []byte("ustar\x00"),
		},
		{
			Name:    "Tar header 'trailerSTAR'",
			Content: []byte("ustar  \x00"),
		},
	}

	for i, c := range tc {
		t.Run(c.Name, func(t *testing.T) {

			// Create a byte slice with the magic bytes at the correct offset
			data := make([]byte, OffsetTar+len(MagicBytesTar[0]))
			copy(data[OffsetTar:], c.Content)

			// Check if IsTar correctly identifies it as a tar file
			if IsTar(data) != true {
				t.Errorf("test case %d failed: %s", i, c.Name)
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

// packTarWithContent creates a tar file with the given content
func packTarWithContent(content []tarContent) []byte {

	// create tar writer
	writeBuffer := bytes.NewBuffer([]byte{})
	tw := tar.NewWriter(writeBuffer)

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

	return writeBuffer.Bytes()
}
