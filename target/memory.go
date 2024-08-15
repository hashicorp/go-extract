package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"time"
)

// NewMemory creates a new in-memory filesystem
func NewMemory() Memory {
	return make(Memory)
}

// CreateFile creates a new file in the in-memory filesystem. The file is created with the given mode.
// If the overwrite flag is set to false and the file already exists, an error is returned. If the overwrite
// flag is set to true, the file is overwritten. The maxSize parameter can be used to limit the size of the file.
// If the file exceeds the maxSize, an error is returned. If the file is created successfully, the number of bytes
// written is returned.
func (m Memory) CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {
	if !overwrite {
		if _, ok := m[path]; ok {
			return 0, fmt.Errorf("file already exists")
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
	if !overwrite {
		if _, ok := m[newName]; ok {
			return fs.ErrExist
		}
	}

	m[newName] = MemoryEntry{
		FileInfo: MemoryFileInfo{name: newName, mode: 0777 | fs.ModeSymlink},
		Data:     []byte(oldName),
	}

	return nil
}

// Open opens a file in the in-memory filesystem. The file is returned as a ReadCloser
// which can be used to read the file contents. If the file does not exist, an error is returned.
// If the file is opened successfully, the ReadCloser is returned.
func (m Memory) Open(path string) (io.ReadCloser, error) {
	if e, ok := m[path]; ok {
		return io.NopCloser(bytes.NewReader(e.Data)), nil
	}
	return nil, fs.ErrNotExist
}

// Lstat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the symlink is returned.
// If the path does not exist, an error is returned.
func (m Memory) Lstat(path string) (fs.FileInfo, error) {
	if e, ok := m[path]; ok {
		return e.FileInfo, nil
	}
	return nil, fs.ErrNotExist
}

// Stat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the target of the symlink is returned.
// If the path does not exist, an error is returned.
func (m Memory) Stat(path string) (fs.FileInfo, error) {
	if e, ok := m[path]; ok {
		if m[path].FileInfo.Mode()&fs.ModeSymlink != 0 {
			linkTarget := string(m[path].Data)
			return m.Stat(linkTarget)
		}
		return e.FileInfo, nil
	}
	return nil, fs.ErrNotExist
}

// Memory is an in-memory filesystem implementation
type Memory map[string]MemoryEntry

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
