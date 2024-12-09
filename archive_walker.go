// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"io"
	"io/fs"
	"time"
)

// archiveWalker is an interface that represents a file walker in an archive
type archiveWalker interface {
	Type() string
	Next() (archiveEntry, error)
}

// archiveEntry is an interface that represents a file in an archive
type archiveEntry interface {
	AccessTime() time.Time
	Gid() int
	IsRegular() bool
	IsDir() bool
	IsSymlink() bool
	Linkname() string
	Mode() fs.FileMode
	ModTime() time.Time
	Name() string
	Open() (io.ReadCloser, error)
	Size() int64
	Sys() interface{}
	Type() fs.FileMode
	Uid() int
}
