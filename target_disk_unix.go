// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package extract

import (
	"time"

	"golang.org/x/sys/unix"
)

// Lchtimes modifies the access and modified timestamps on a target path
// This capability is only available on unix as of now.
func Lchtimes(path string, atime, mtime time.Time) error {

	return unix.Lutimes(path, []unix.Timeval{
		{Sec: atime.Unix(), Usec: int32(atime.Nanosecond() / 1e6 % 1e6)},
		{Sec: mtime.Unix(), Usec: int32(mtime.Nanosecond() / 1e6 % 1e6)},
	})
}

// CanMaintainSymlinkTimestamps determines whether is is possible to change
// timestamps on symlinks for the the current platform. For regular files
// and directories, attempts are made to restore permissions and timestamps
// after extraction. But for symbolic links, go's cross-platform
// packages (Chmod and Chtimes) are not capable of changing symlink info
// because those methods follow the symlinks. However, a platform-dependent option
// is provided for unix (see Lchtimes)
func CanMaintainSymlinkTimestamps() bool {
	return true
}
