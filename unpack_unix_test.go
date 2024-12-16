// Copyright (c) HashiCorp, Inc.
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
	uid, gid := 503, 20
	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	testCases := []struct {
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
				{Name: "test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: 0, Gid: 0},
				{Name: "sub", Mode: fs.ModeDir | 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "sub/test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "link", Mode: fs.ModeSymlink | 0777, Linktarget: "sub/test", AccessTime: baseTime, ModTime: baseTime},
			},
			packer: packTar,
		},
		{
			name: "zip",
			contents: []archiveContent{
				{Name: "test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "sub", Mode: fs.ModeDir | 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "sub/test", Content: []byte("hello world"), Mode: 0644, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "link", Mode: fs.ModeSymlink | 0777, Linktarget: "sub/test", AccessTime: baseTime, ModTime: baseTime},
			},
			doesNotSupportOwner: true,
			packer:              packZip,
		},
		{
			name:                  "rar",
			contents:              contentsRar2,
			doesNotSupportModTime: true,
			doesNotSupportOwner:   true,
			packer:                packRar2,
		},
		{
			name:                "7z",
			contents:            contents7z2,
			doesNotSupportOwner: true,
			packer:              pack7z2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ctx = context.Background()
				dst = t.TempDir()
				src = asIoReader(t, tc.packer(t, tc.contents))
				cfg = extract.NewConfig(extract.WithPreserveFileAttributes(true))
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
				if os.Getuid() != 0 || tc.doesNotSupportOwner {
					continue
				}
				if stat.Sys().(*syscall.Stat_t).Uid != uint32(c.Uid) {
					t.Fatalf("expected uid %d, got %d, file %s", c.Uid, stat.Sys().(*syscall.Stat_t).Uid, c.Name)
				}
			}
		})
	}
}
