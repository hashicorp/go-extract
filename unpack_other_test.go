// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !unix

package extract_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/go-extract"
)

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
					var (
						path = filepath.Join(t.TempDir(), "test")
						mode = fs.FileMode(u | g | o)
					)
					if err := func() error {
						if dir {
							return os.Mkdir(path, mode)
						}
						return os.WriteFile(path, []byte("foobar content"), mode)
					}(); err != nil {
						t.Fatalf("error creating test resource: %s", err)
					}
					stat, err := os.Stat(path)
					if err != nil {
						t.Fatalf("error getting file stats: %s", err)
					}
					calculated := toWindowsFileMode(dir, mode)
					if stat.Mode().Perm() != calculated.Perm() {
						t.Errorf("toWindowsFileMode(%t, %s) calculated mode mode %s, but actual windows mode: %s", dir, mode, calculated.Perm(), stat.Mode().Perm())
					}
				}
			}
		}
	}
}

func TestWithCustomMode(t *testing.T) {

	if runtime.GOOS != "windows" {
		t.Skip("test only runs on Windows")
	}

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
			if test.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for name, expectedMode := range test.expected {
				stat, err := os.Stat(filepath.Join(tmp, name))
				if err != nil {
					t.Fatalf("error getting file stats: %s", err)
				}

				if runtime.GOOS == "windows" {
					if stat.IsDir() {
						continue // Skip directory checks on Windows
					}
					expectedMode = toWindowsFileMode(stat.IsDir(), expectedMode)
				} else {
					expectedMode &= ^umask // Adjust for umask on non-Windows systems
				}

				if stat.Mode().Perm() != expectedMode.Perm() {
					t.Fatalf("expected directory/file to have mode %s, but got: %s", expectedMode.Perm(), stat.Mode().Perm())
				}
			}
		})
	}
}
