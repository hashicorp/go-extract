// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build unix

package extract

import (
	"io/fs"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// Chown changes the numeric uid and gid of the named file.
func (d *TargetDisk) Chown(name string, uid, gid int) error {
	if err := os.Lchown(name, uid, gid); err != nil {
		return fmt.Errorf("chown failed: %w", err)
	}
	return nil
}

// lchtimes modifies the access and modified timestamps on a target path
// This capability is only available on unix as of now.
func lchtimes(path string, atime, mtime time.Time) error {
	return unix.Lutimes(path, []unix.Timeval{
		unixTimeval(atime),
		unixTimeval(mtime),
	})
}

// unixTimeval converts a time.Time to a unix.Timeval. Note that it always rounds
// up to the nearest microsecond, so even one nanosecond past the previous nanosecond
// will be rounded up to the next microsecond.
// See the implementation of unix.NsecToTimeval for details on how this happens.
func unixTimeval(t time.Time) unix.Timeval {
	return unix.NsecToTimeval(t.UnixNano())
}

// canMaintainSymlinkTimestamps determines whether is is possible to change
// timestamps on symlinks for the the current platform. For regular files
// and directories, attempts are made to restore permissions and timestamps
// after extraction. But for symbolic links, go's cross-platform
// packages (Chmod and Chtimes) are not capable of changing symlink info
// because those methods follow the symlinks. However, a platform-dependent option
// is provided for unix (see Lchtimes)
const canMaintainSymlinkTimestamps = true
