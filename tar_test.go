package extract_test

import (
	"archive/tar"
	"context"
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/go-extract"
)

func TestTarUnpackNew(t *testing.T) {
	// generate cancled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name        string
		content     []byte
		opts        []extract.ConfigOption
		expectError bool
		ctx         context.Context
	}{
		{
			name:        "unpack normal tar",
			content:     packTar(t, []archiveContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			expectError: false,
		},
		{
			name:        "unpack normal tar, but pattern mismatch",
			content:     packTar(t, []archiveContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			opts:        []extract.ConfigOption{extract.WithPatterns("*foo")},
			expectError: false,
		},
		{
			name:        "unpack normal tar, but context canceled",
			content:     packTar(t, []archiveContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			ctx:         canceledCtx,
			expectError: true,
		},
		{
			name: "unpack normal tar with 5 files",
			content: packTar(t, []archiveContent{
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
			content: packTar(t, []archiveContent{
				{Content: []byte("foobar content"), Name: "test1", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test2", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test3", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test4", Mode: 0640, Filetype: tar.TypeReg},
				{Content: []byte("foobar content"), Name: "test5", Mode: 0640, Filetype: tar.TypeReg},
			}),
			opts:        []extract.ConfigOption{extract.WithMaxFiles(4)},
			expectError: true,
		},
		{
			name:        "unpack normal tar, but extraction size exceeded",
			content:     packTar(t, []archiveContent{{Content: []byte("foobar content"), Name: "test", Mode: 0640, Filetype: tar.TypeReg}}),
			opts:        []extract.ConfigOption{extract.WithMaxExtractionSize(1)},
			expectError: true,
		},
		{
			name:        "unpack malicious tar, with traversal",
			content:     packTar(t, []archiveContent{{Content: []byte("foobar content"), Name: "../test", Mode: 0640, Filetype: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink",
			content:     packTar(t, []archiveContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			expectError: false,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     packTar(t, []archiveContent{{Name: "../test", Filetype: tar.TypeDir}}),
			expectError: true,
		},
		{
			name:        "unpack tar with traversal in directory",
			content:     packTar(t, []archiveContent{{Name: "../test", Filetype: tar.TypeDir}}),
			opts:        []extract.ConfigOption{extract.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name:        "unpack normal tar with traversal symlink",
			content:     packTar(t, []archiveContent{{Name: "foo", Linktarget: "../bar", Filetype: tar.TypeLink}}),
			expectError: true,
		},
		{
			name:        "unpack normal tar with symlink, but symlinks are denied",
			content:     packTar(t, []archiveContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			opts:        []extract.ConfigOption{extract.WithDenySymlinkExtraction(true)},
			expectError: true,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on error",
			content: packTar(t, []archiveContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			opts: []extract.ConfigOption{
				extract.WithDenySymlinkExtraction(true),
				extract.WithContinueOnError(true),
			},
			expectError: false,
		},
		{
			name:    "unpack normal tar with symlink, but symlinks are denied, but continue on unsupported files",
			content: packTar(t, []archiveContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "testTarget"}}),
			opts: []extract.ConfigOption{
				extract.WithDenySymlinkExtraction(true),
				extract.WithContinueOnUnsupportedFiles(true),
			},
			expectError: false,
		},
		{
			name:        "unpack normal tar with absolute path in symlink",
			content:     packTar(t, []archiveContent{{Name: "testLink", Filetype: tar.TypeSymlink, Linktarget: "/absolute-target"}}),
			expectError: runtime.GOOS != "windows",
		},
		{
			name:        "malicious tar with symlink name path traversal",
			content:     packTar(t, []archiveContent{{Name: "../testLink", Filetype: tar.TypeSymlink, Linktarget: "target"}}),
			expectError: true,
		},
		{
			name:        "malicious tar with .. as filename",
			content:     packTar(t, []archiveContent{{Content: []byte("foobar content"), Name: "..", Filetype: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "malicious tar with . as filename",
			content:     packTar(t, []archiveContent{{Content: []byte("foobar content"), Name: ".", Filetype: tar.TypeReg}}),
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype",
			content:     packTar(t, []archiveContent{{Name: "fifo", Filetype: tar.TypeFifo}}),
			expectError: true,
		},
		{
			name:        "malicious tar with FIFO filetype, but continue on error",
			content:     packTar(t, []archiveContent{{Name: "fifo", Filetype: tar.TypeFifo}}),
			opts:        []extract.ConfigOption{extract.WithContinueOnUnsupportedFiles(true)},
			expectError: false,
		},
		{
			name: "malicious tar with zip slip attack",
			content: packTar(t, []archiveContent{
				{Name: "sub/to-parent", Filetype: tar.TypeSymlink, Linktarget: "../"},
				{Name: "sub/to-parent/one-above", Filetype: tar.TypeSymlink, Linktarget: "../"},
			}),
			expectError: true,
		},
		{
			name:        "tar with legit git pax_global_header",
			content:     packTar(t, []archiveContent{{Content: []byte(""), Name: "pax_global_header", Filetype: tar.TypeXGlobalHeader}}),
			expectError: false,
		},
		{
			name: "absolute path in filename (windows), fails bc of ':' in path",
			content: packTar(t, []archiveContent{
				{Content: []byte("foobar content"), Name: "s:\\absolute-path", Mode: 0640, Filetype: tar.TypeReg},
			}),
			expectError: runtime.GOOS == "windows",
		},
		{
			name: "absolute path in filename",
			content: packTar(t, []archiveContent{
				{Content: []byte("foobar content"), Name: "/absolute-path", Mode: 0640, Filetype: tar.TypeReg},
			}),
			expectError: false,
		},
		{
			name: "extract a directory",
			content: packTar(t, []archiveContent{
				{Name: "test", Filetype: tar.TypeDir},
			}),
			expectError: false,
		},
		{
			name: "extract a file with traversal, but continue on error",
			content: packTar(t, []archiveContent{
				{Content: []byte("foobar content"), Name: "../test", Mode: 0640, Filetype: tar.TypeReg},
			}),
			opts:        []extract.ConfigOption{extract.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "extract a symlink with traversal, but continue on error",
			content: packTar(t, []archiveContent{
				{Name: "foo", Linktarget: "../bar", Filetype: tar.TypeSymlink},
			}),
			opts:        []extract.ConfigOption{extract.WithContinueOnError(true)},
			expectError: false,
		},
		{
			name: "tar with hard link, with error, but continue on error",
			content: packTar(t, []archiveContent{
				{Name: "testLink", Filetype: tar.TypeLink, Linktarget: "testTarget"},
			}),
			opts:        []extract.ConfigOption{extract.WithContinueOnError(true)},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new target
			testingTarget := extract.NewDisk()

			// create testing directory
			testDir := t.TempDir()

			if test.ctx == nil {
				test.ctx = context.Background()
			}
			ctx := test.ctx

			// perform actual tests
			input := newTestFile(filepath.Join(testDir, "test.tar"), test.content)
			want := test.expectError
			err := extract.UnpackTar(ctx, testingTarget, testDir, input, extract.NewConfig(test.opts...))
			defer input.(io.Closer).Close()
			got := err != nil
			if got != want {
				t.Errorf("error = %v, wantErr %v", err, want)
			}
		})
	}
}

// func Test_isTar(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		content []byte
// 	}{
// 		{
// 			name:    "Tar header 'magicGNU/versionGNU'",
// 			content: []byte("ustar\x00tar\x00"),
// 		},
// 		{
// 			name:    "Tar header 'magicUSTAR/versionUSTAR'",
// 			content: []byte("ustar\x00"),
// 		},
// 		{
// 			name:    "Tar header 'trailerSTAR'",
// 			content: []byte("ustar  \x00"),
// 		},
// 	}

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			// Create a byte slice with the magic bytes at the correct offset
// 			data := make([]byte, offsetTar+len(magicBytesTar[0]))
// 			copy(data[extract.offsetTar:], test.content)

// 			// Check if IsTar correctly identifies it as a tar file
// 			if !extract.IsTar(data) {
// 				t.Logf("expected data to be identified as a tar file")
// 			}
// 		})
// 	}
// }
