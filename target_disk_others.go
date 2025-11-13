// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !unix

package extract

import (
	"fmt"
	"runtime"
	"time"
)

// lchtimes modifies the access and modified timestamps on a target path
// This capability is only available on unix as of now.
func lchtimes(_ string, _, _ time.Time) error {
	return fmt.Errorf("Lchtimes is not supported on this platform (%s)", runtime.GOOS)
}

// canMaintainSymlinkTimestamps determines whether is is possible to change
// timestamps on symlinks for the the current platform. For regular files
// and directories, attempts are made to restore permissions and timestamps
// after extraction. But for symbolic links, go's cross-platform
// packages (Chmod and Chtimes) are not capable of changing symlink info
// because those methods follow the symlinks. However, a platform-dependent option
// is provided for unix (see Lchtimes)
const canMaintainSymlinkTimestamps = false

// Chown changes the numeric uid and gid of the named file.
func (d *TargetDisk) Chown(name string, uid, gid int) error {
	return fmt.Errorf("Chown is not supported on this platform (%s)", runtime.GOOS)
}
