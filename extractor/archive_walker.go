package extractor

import (
	"io"
	"io/fs"
)

// archiveWalker is an interface that represents a file walker in an archive
type archiveWalker interface {
	Type() string
	Next() (archiveEntry, error)
}

// archiveEntry is an interface that represents a file in an archive
type archiveEntry interface {
	Mode() fs.FileMode
	Type() fs.FileMode
	Name() string
	Linkname() string
	Size() int64
	Open() (io.ReadCloser, error)
	IsRegular() bool
	IsDir() bool
	IsSymlink() bool
}
