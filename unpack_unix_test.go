// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build unix

package extract_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/go-extract"
)

func TestUnpackWithPreserveFileAttributes(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ctx = context.Background()
				dst = t.TempDir()
				src = asIoReader(t, tc.packer(t, tc.contents))
				cfg = extract.NewConfig()
			)
			if err := extract.Unpack(ctx, dst, src, cfg); err != nil {
				t.Fatalf("error unpacking archive: %v", err)
			}
			for _, c := range tc.contents {
				path := filepath.Join(dst, c.Name)
				stat, err := os.Lstat(path)
				if err != nil {
					t.Fatalf("error getting file stats: %v", err)
				}
				if !(c.Mode&fs.ModeSymlink != 0) { // skip symlink checks
					if stat.Mode().Perm() != c.Mode.Perm() {
						t.Fatalf("expected file mode %v, got %v, file %s", c.Mode.Perm(), stat.Mode().Perm(), c.Name)
					}
				}
				if tc.doesNotSupportModTime {
					continue
				}
				modTimeDiff := abs(stat.ModTime().UnixNano() - c.ModTime.UnixNano())
				if modTimeDiff >= int64(time.Microsecond) {
					t.Fatalf("expected mod time %v, got %v, file %s, diff %v", c.ModTime, stat.ModTime(), c.Name, modTimeDiff)
				}
			}
		})
	}
}

func TestUnpackWithPreserveOwnershipAsNonRoot(t *testing.T) {

	if os.Getuid() == 0 {
		t.Skip("test requires non-root privileges")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// skip test if the archive does not store ownership information
			if tc.doesNotSupportOwner {
				t.Skipf("archive %s does not store ownership information", tc.name)
			}

			var (
				ctx = context.Background()
				dst = t.TempDir()
				src = asIoReader(t, tc.packer(t, tc.contents))
				cfg = extract.NewConfig(extract.WithPreserveOwner(true))
			)

			// Unpack should fail if the user is not root and the uid/gid
			// in the archive is different from the current user (only
			// if the archive supports owner information)
			err := extract.Unpack(ctx, dst, src, cfg)

			// chown will only fail if the user is not root and the
			// uid/gid in the archive is different from the current user
			if err == nil {
				t.Fatalf("got nil error; want permissions error")
			}
		})
	}
}

func TestUnpackWithPreserveOwnershipAsRoot(t *testing.T) {

	if os.Getuid() != 0 {
		t.Skip("test requires root privileges")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// skip test if the archive does not store ownership information
			if tc.doesNotSupportOwner {
				t.Skipf("archive type %s does not store ownership information", tc.name)
			}

			var (
				ctx = context.Background()
				dst = t.TempDir()
				src = asIoReader(t, tc.packer(t, tc.contents))
				cfg = extract.NewConfig(extract.WithPreserveOwner(true))
			)

			if err := extract.Unpack(ctx, dst, src, cfg); err != nil {
				t.Fatalf("error unpacking archive: %v", err)
			}

			// check ownership of files
			expectUidMatch := !tc.invalidUidGid
			for _, c := range tc.contents {
				path := filepath.Join(dst, c.Name)
				stat, err := os.Lstat(path)
				if err != nil {
					t.Fatalf("error getting file stats: %v", err)
				}
				uidMatch := c.Uid == int(stat.Sys().(*syscall.Stat_t).Uid)
				if expectUidMatch != uidMatch {
					t.Fatalf("expected uid %d, got %d, file %s", c.Uid, stat.Sys().(*syscall.Stat_t).Uid, c.Name)
				}
			}
		})
	}
}

func TestWithCustomMode(t *testing.T) {
	umask := sniffUmask(t)

	tests := []struct {
		name     string
		data     []byte
		dst      string
		cfg      *extract.Config
		expected map[string]fs.FileMode
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
				extract.WithCustomCreateDirMode(fs.FileMode(0757 & ^umask)), // 493 & ^umask
			),
			expected: map[string]fs.FileMode{
				"sub":      fs.FileMode(0757 & ^umask), // 493 & ^umask <-- implicit created dir
				"sub/file": fs.FileMode(0644),          // 420
			},
		},
		{
			name: "decompress with custom mode",
			data: compressGzip(t, []byte("foobar content")),
			dst:  "out", // specify decompressed file name
			cfg: extract.NewConfig(
				extract.WithCustomDecompressFileMode(fs.FileMode(0666)), // 438 + umask is applied while file creation
			),
			expected: map[string]fs.FileMode{
				"out": 0666 & ^umask, // 438 & ^umask
			},
		},
		{
			name: "dir with 0755 and file with 0777",
			data: compressGzip(t, []byte("foobar content")),
			dst:  "foo/out",
			cfg: extract.NewConfig(
				extract.WithCreateDestination(true),                     // create destination^
				extract.WithCustomCreateDirMode(fs.FileMode(0750)),      // 488 + umask is applied while dir creation
				extract.WithCustomDecompressFileMode(fs.FileMode(0777)), // 511 + umask is applied while file creation
			),
			expected: map[string]fs.FileMode{
				"foo":     fs.FileMode(0750 & ^umask), // 488 & ^umask
				"foo/out": fs.FileMode(0777 & ^umask), // 511 & ^umask
			},
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
				extract.WithCustomCreateDirMode(fs.FileMode(0777)), // 511 + umask is applied while dir creation
			),
			expected: map[string]fs.FileMode{
				"sub":      fs.FileMode(0777 & ^umask), // 511
				"sub/file": fs.FileMode(0777),          // 511 <-- is preserved from the archive and umask is not applied
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
		{
			name: "dir with 777 and file with 777 but no file attribute mode preservation",
			data: compressGzip(t, packTar(t, []archiveContent{
				{
					Name: "file",
					Mode: fs.FileMode(0777), // 511
				},
				{
					Name: "dir",
					Mode: fs.ModeDir | 0777, // 511
				},
			})),
			cfg: extract.NewConfig(extract.WithDropFileAttributes(true)),
			expected: map[string]fs.FileMode{
				"file": fs.FileMode(0777 & ^umask), // 438
				"dir":  fs.FileMode(0777 & ^umask), // 438
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.cfg == nil {
				test.cfg = extract.NewConfig()
			}
			var (
				ctx = context.Background()
				tmp = t.TempDir()
				dst = filepath.Join(tmp, test.dst)
				src = asIoReader(t, test.data)
				cfg = test.cfg
			)
			err := extract.Unpack(ctx, dst, src, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for name, expectedMode := range test.expected {
				stat, err := os.Stat(filepath.Join(tmp, name))
				if err != nil {
					t.Fatalf("error getting file stats: %s", err)
				}
				if stat.Mode().Perm() != expectedMode.Perm() {
					t.Fatalf("expected %s to have mode %s, but got: %s", name, expectedMode.Perm(), stat.Mode().Perm())
				}
			}
		})
	}
}
