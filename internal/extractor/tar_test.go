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
)

// TestTarUnpack implements test cases
func TestTarUnpackNew(t *testing.T) {
	// generate cancled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name        string
		content     []byte
		opts        []config.ConfigOption
		expectError bool
		ctx         context.Context
	}{
		{
			name:        "unpack normal tar",
			content:     packTarWithContent(t, []tarContent{{content: []byte("foobar content"), name: "test", mode: 0640, fileType: tar.TypeReg}}),
			expectError: false,
		},
		{
			name:        "unpack normal tar, but pattern mismatch",
			content:     packTarWithContent(t, []tarContent{{content: []byte("foobar content"), name: "test", mode: 0640, fileType: tar.TypeReg}}),
			opts:        []config.ConfigOption{config.WithPatterns("*foo")},
			expectError: false,
		},
		{
			name:        "unpack normal tar, but context canceled",
			content:     packTarWithContent(t, []tarContent{{content: []byte("foobar content"), name: "test", mode: 0640, fileType: tar.TypeReg}}),
			ctx:         canceledCtx,
			expectError: true,
		},
		{
			name: "unpack normal tar with 5 files",
			content: packTarWithContent(t, []tarContent{
				{content: []byte("foobar content"), name: "test1", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test2", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test3", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test4", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test5", mode: 0640, fileType: tar.TypeReg},
			}),
			expectError: false,
		},
		{
			name: "unpack normal tar with 5 files, but file limit",
			content: packTarWithContent(t, []tarContent{
				{content: []byte("foobar content"), name: "test1", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test2", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test3", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test4", mode: 0640, fileType: tar.TypeReg},
				{content: []byte("foobar content"), name: "test5", mode: 0640, fileType: tar.TypeReg},
			}),
			opts:        []config.ConfigOption{config.WithMaxFiles(4)},
			expectError: true,
		},
		{
			name:        "unpack normal tar, but extraction size exceeded",
			content:     packTarWithContent(t, []tarContent{{content: []byte("foobar content"), name: "test", mode: 0640, fileType: tar.TypeReg}}),
			opts:        []config.ConfigOption{config.WithMaxExtractionSize(1)},
			expectError: true,
		},
		{
			name:        "unpack malicious tar, with traversal",
			content:     packTarWithContent(t, []tarContent{{content: []byte("foobar content"), name: "../test", mode: 0640, fileType: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink",
			content:     packTarWithContent(t, []tarContent{{name: "testLink", fileType: tar.TypeSymlink, linktarget: "testTarget"}}),
			expectError: false,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     packTarWithContent(t, []tarContent{{name: "../test", fileType: tar.TypeDir}}),
			expectError: true,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     packTarWithContent(t, []tarContent{{name: "../test", fileType: tar.TypeDir}}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name:        "unpack normal tar with traversal symlink",
			content:     packTarWithContent(t, []tarContent{{name: "foo", linktarget: "../bar", fileType: tar.TypeLink}}),
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink, but symlinks are denied",
			content:     packTarWithContent(t, []tarContent{{name: "testLink", fileType: tar.TypeSymlink, linktarget: "testTarget"}}),
			opts:        []config.ConfigOption{config.WithDenySymlinkExtraction(true)},
			expectError: true,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on error",
			content: packTarWithContent(t, []tarContent{{name: "testLink", fileType: tar.TypeSymlink, linktarget: "testTarget"}}),
			opts: []config.ConfigOption{
				config.WithDenySymlinkExtraction(true),
				config.WithContinueOnError(true),
			},
			expectError: false,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on unsupported files",
			content: packTarWithContent(t, []tarContent{{name: "testLink", fileType: tar.TypeSymlink, linktarget: "testTarget"}}),
			opts: []config.ConfigOption{
				config.WithDenySymlinkExtraction(true),
				config.WithContinueOnUnsupportedFiles(true),
			},
			expectError: false,
		},
		{
			name:        "unpack normal tar with absolute path in symlink",
			content:     packTarWithContent(t, []tarContent{{name: "testLink", fileType: tar.TypeSymlink, linktarget: "/absolute-target"}}),
			expectError: runtime.GOOS != "windows",
		},
		{
			name:        "malicious tar with symlink name path traversal",
			content:     packTarWithContent(t, []tarContent{{name: "../testLink", fileType: tar.TypeSymlink, linktarget: "target"}}),
			expectError: true,
		},
		{
			name:        "malicious tar with .. as filename",
			content:     packTarWithContent(t, []tarContent{{content: []byte("foobar content"), name: "..", fileType: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "malicious tar with . as filename",
			content:     packTarWithContent(t, []tarContent{{content: []byte("foobar content"), name: ".", fileType: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype",
			content:     packTarWithContent(t, []tarContent{{name: "fifo", fileType: tar.TypeFifo}}),
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype, but continue on error",
			content:     packTarWithContent(t, []tarContent{{name: "fifo", fileType: tar.TypeFifo}}),
			opts:        []config.ConfigOption{config.WithContinueOnUnsupportedFiles(true)},
			expectError: false,
		},
		{
			name: "malicious tar with zip slip attack",
			content: packTarWithContent(t, []tarContent{
				{name: "sub/to-parent", fileType: tar.TypeSymlink, linktarget: "../"},
				{name: "sub/to-parent/one-above", fileType: tar.TypeSymlink, linktarget: "../"},
			}),
			expectError: true,
		},
		{
			name:        "tar with legit git pax_global_header",
			content:     packTarWithContent(t, []tarContent{{content: []byte(""), name: "pax_global_header", fileType: tar.TypeXGlobalHeader}}),
			expectError: false,
		},
		{
			name: "absolute path in filename (windows), fails bc of ':' in path",
			content: packTarWithContent(t, []tarContent{
				{content: []byte("foobar content"), name: "s:\\absolute-path", mode: 0640, fileType: tar.TypeReg},
			}),
			expectError: runtime.GOOS == "windows",
		},
		{
			name: "absolute path in filename",
			content: packTarWithContent(t, []tarContent{
				{content: []byte("foobar content"), name: "/absolute-path", mode: 0640, fileType: tar.TypeReg},
			}),
			expectError: false,
		},
		{
			name: "extract a directory",
			content: packTarWithContent(t, []tarContent{
				{name: "test", fileType: tar.TypeDir},
			}),
			expectError: false,
		},
		{
			name: "extract a file with traversal, but continue on error",
			content: packTarWithContent(t, []tarContent{
				{content: []byte("foobar content"), name: "../test", mode: 0640, fileType: tar.TypeReg},
			}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "extract a symlink with traversal, but continue on error",
			content: packTarWithContent(t, []tarContent{
				{name: "foo", linktarget: "../bar", fileType: tar.TypeSymlink},
			}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "tar with hard link, with error, but continue on error",
			content: packTarWithContent(t, []tarContent{
				{name: "testLink", fileType: tar.TypeLink, linktarget: "testTarget"},
			}),
			opts:        []config.ConfigOption{config.WithContinueOnError(true)},
			expectError: false,
		},
	}

	// run cases
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Create a new target
			testingTarget := NewOS()

			// create testing directory
			testDir := t.TempDir()

			if test.ctx == nil {
				test.ctx = context.Background()
			}
			ctx := test.ctx

			// perform actual tests
			input := newTestFile(filepath.Join(testDir, "test.tar"), test.content)
			want := test.expectError
			err := UnpackTar(ctx, testingTarget, testDir, input, config.NewConfig(test.opts...))
			defer input.(io.Closer).Close()
			got := err != nil
			if got != want {
				t.Errorf("error = %v, wantErr %v", err, want)
			}
		})
	}
}

func Test_isTar(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "Tar header 'magicGNU/versionGNU'",
			content: []byte("ustar\x00tar\x00"),
		},
		{
			name:    "Tar header 'magicUSTAR/versionUSTAR'",
			content: []byte("ustar\x00"),
		},
		{
			name:    "Tar header 'trailerSTAR'",
			content: []byte("ustar  \x00"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a byte slice with the magic bytes at the correct offset
			data := make([]byte, offsetTar+len(magicBytesTar[0]))
			copy(data[offsetTar:], test.content)

			// Check if IsTar correctly identifies it as a tar file
			if !isTar(data) {
				t.Logf("expected data to be identified as a tar file")
			}
		})
	}
}

// tarContent is a struct to store the content of a tar file
type tarContent struct {
	content    []byte
	linktarget string
	mode       os.FileMode
	name       string
	fileType   byte
}

// packTarWithContent creates a tar file with the given content
func packTarWithContent(t *testing.T, content []tarContent) []byte {
	t.Helper()

	// create tar writer
	writeBuffer := bytes.NewBuffer([]byte{})
	tw := tar.NewWriter(writeBuffer)

	// write content
	for _, c := range content {
		// create header
		hdr := &tar.Header{
			Name:     c.name,
			Mode:     int64(c.mode),
			Size:     int64(len(c.content)),
			Linkname: c.linktarget,
			Typeflag: c.fileType,
		}

		// write header
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("error writing tar header: %v", err)
		}

		// write data
		if _, err := tw.Write(c.content); err != nil {
			t.Fatalf("error writing tar data: %v", err)
		}
	}

	// close tar writer
	if err := tw.Close(); err != nil {
		t.Fatalf("error closing tar writer: %v", err)
	}

	return writeBuffer.Bytes()
}
