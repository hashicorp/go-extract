package target

import (
	"io"
	"io/fs"
)

// Target specifies all function that are needed to be implemented to extract contents from an archive
type Target interface {

	// CreateFile creates a file at the specified path with src as content. The mode parameter is the file mode that
	// should be set on the file. If the file already exists and overwrite is false, an error should be returned. If the
	// file does not exist, it should be created. The size of the file should not exceed maxSize. If the file is created
	// successfully, the number of bytes written should be returned. If an error occurs, the number of bytes written
	// should be returned along with the error. If maxSize < 0, the file size is not limited.
	CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error)

	// CreateDir creates at the specified path with the specified mode. If the directory already exists, nothing is done.
	// The function returns an error if there's a problem creating the directory. If the function completes successfully,
	// it returns nil.
	CreateDir(path string, mode fs.FileMode) error

	// CreateSymlink creates a symbolic link from newname to oldname. If newname already exists and overwrite is false,
	// the function returns an error. If newname already exists and overwrite is true, the function may overwrite the
	// existing symlink.
	CreateSymlink(oldname string, newname string, overwrite bool) error

	// Lstat returns the FileInfo for the specified path. If the path does not exist, the function should return an error.
	// If the path exists, the function should return the FileInfo for the path. The function should return an error if
	// there's a problem getting the FileInfo.
	Lstat(path string) (fs.FileInfo, error)
}
