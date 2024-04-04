package target

import (
	"io"
	"io/fs"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {

	// CreateFile is used to create a file with name and content
	CreateFile(name string, content io.Reader, perm fs.FileMode, overwrite bool, maxSize int64) (int64, error)

	// CreateDir creates directory with name and perm
	CreateDir(name string, perm fs.FileMode) error

	// CreateSymlink creates symlink name with destination linkTarget in dstDir
	CreateSymlink(name string, target string, overwrite bool) error

	// Lstat returns the FileInfo of the file at path
	Lstat(path string) (fs.FileInfo, error)
}
