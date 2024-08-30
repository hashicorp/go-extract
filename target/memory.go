package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"time"
)

// Memory is an in-memory filesystem implementation. It is a map of file paths to MemoryEntry.
// The MemoryEntry contains the file information and the file data.
// The Memory filesystem can be used to create, read, and write files in memory. It can also be
// used to create directories and symlinks. Permissions on entries (owner, group, others) are
// not enforced. Entries can be accessed by the path as a key in the map, or by calling
// the m.Open(<path>) function.
type Memory map[string]MemoryEntry

// NewMemory creates a new in-memory filesystem.
func NewMemory() Target {
	return make(Memory)
}

// CreateFile creates a new file in the in-memory filesystem. The file is created with the given mode.
// If the overwrite flag is set to false and the file already exists, an error is returned. If the overwrite
// flag is set to true, the file is overwritten. The maxSize parameter can be used to limit the size of the file.
// If the file exceeds the maxSize, an error is returned. If the file is created successfully, the number of bytes
// written is returned.
func (m Memory) CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {
	if !fs.ValidPath(path) {
		return 0, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if !overwrite {
		if _, ok := m[path]; ok {
			return 0, fmt.Errorf("%w: %s", fs.ErrExist, path)
		}
	}

	// create byte buffered writer
	var buf bytes.Buffer
	w := limitWriter(&buf, maxSize)

	// write to buffer
	n, err := io.Copy(w, src)
	if err != nil {
		return n, err
	}

	// create entry
	m[path] = MemoryEntry{
		FileInfo: MemoryFileInfo{name: path, size: n, mode: mode.Perm(), modTime: time.Now()},
		Data:     buf.Bytes(),
	}

	// return number of bytes written
	return n, nil
}

// CreateDir creates a new directory in the in-memory filesystem.
// If the directory already exists, nothing is done. If the directory does not exist, it is created.
// The directory is created with the given mode. If the directory is created successfully, nil is returned.
func (m Memory) CreateDir(path string, mode fs.FileMode) error {
	if !fs.ValidPath(path) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}

	if _, ok := m[path]; ok {
		return nil
	}

	m[path] = MemoryEntry{
		FileInfo: MemoryFileInfo{name: path, mode: mode.Perm() | fs.ModeDir},
	}

	return nil
}

// CreateSymlink creates a new symlink in the in-memory filesystem.
// If the overwrite flag is set to false and the symlink already exists, an error is returned.
// If the overwrite flag is set to true, the symlink is overwritten. If the symlink is created successfully, nil is returned.
func (m Memory) CreateSymlink(oldName string, newName string, overwrite bool) error {
	if !fs.ValidPath(newName) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, newName)
	}
	if !overwrite {
		if _, ok := m[newName]; ok {
			return fmt.Errorf("%w: %s", fs.ErrExist, newName)
		}
	}

	m[newName] = MemoryEntry{
		FileInfo: MemoryFileInfo{name: newName, mode: 0777 | fs.ModeSymlink},
		Data:     []byte(oldName),
	}

	return nil
}

// Open opens the named file for reading. If successful, the file is returned
// as an [io.ReadCloser] which can be used to read the file contents. If the
// file is  a symlink, the target of the symlink is opened. If the file does not
// exist, or is a directory, an error is returned.
func (m Memory) Open(path string) (io.ReadCloser, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}

	// get entry
	e, ok := m[path]

	// file does not exist
	if !ok {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
	}

	// handle directory
	if e.FileInfo.Mode()&fs.ModeDir != 0 {
		return nil, fmt.Errorf("cannot open directory")
	}

	// handle symlink
	if e.FileInfo.Mode()&fs.ModeSymlink != 0 {
		linkTarget := string(e.Data)
		return m.Open(linkTarget)
	}

	// return file data
	return io.NopCloser(bytes.NewReader(e.Data)), nil

}

// Lstat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the symlink is returned.
// If the path does not exist, an error is returned.
func (m Memory) Lstat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m[path]; ok {
		return e.FileInfo, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Stat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the target of the symlink is returned.
// If the path does not exist, an error is returned.
func (m Memory) Stat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m[path]; ok {
		if m[path].FileInfo.Mode()&fs.ModeSymlink != 0 {
			linkTarget := string(m[path].Data)
			return m.Stat(linkTarget)
		}
		return e.FileInfo, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Readlink returns the target of the symlink at the given path. If the path is not a symlink, an error is returned.
// If the path does not exist, an error is returned. If the symlink exists, the target of the symlink is returned.
func (m Memory) Readlink(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m[path]; ok {
		if e.FileInfo.Mode()&fs.ModeSymlink != 0 {
			return string(e.Data), nil
		}
		return "", fmt.Errorf("not a symlink: %w: %s", fs.ErrInvalid, path)
	}
	return "", fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Remove removes the entry at the given path. If the path does not exist, an error is returned.
func (m Memory) Remove(path string) error {
	if !fs.ValidPath(path) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	delete(m, path)
	return nil
}

// MemoryEntry is an entry in the in-memory filesystem
type MemoryEntry struct {
	FileInfo fs.FileInfo
	Data     []byte
}

// MemoryFileInfo is a FileInfo implementation for the in-memory filesystem
type MemoryFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

// Name returns the name of the file
func (fi MemoryFileInfo) Name() string {
	return fi.name
}

// Size returns the size of the file
func (fi MemoryFileInfo) Size() int64 {
	return fi.size
}

// Mode returns the mode of the file
func (fi MemoryFileInfo) Mode() fs.FileMode {
	return fi.mode
}

// ModTime returns the modification time of the file
func (fi MemoryFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir returns true if the file is a directory
func (fi MemoryFileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

// Sys returns the underlying data source (nil for in-memory filesystem)
func (fi MemoryFileInfo) Sys() any {
	return nil
}
