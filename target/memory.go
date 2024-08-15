package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"time"
)

func NewMemory() Memory {
	return make(Memory)
}

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

func (m Memory) CreateDir(path string, mode fs.FileMode) error {
	if _, ok := m[path]; ok {
		return nil
	}

	m[path] = MemoryEntry{
		FileInfo: MemoryFileInfo{name: path, mode: mode.Perm() | fs.ModeDir},
	}

	return nil
}

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

func (m Memory) Open(path string) (io.ReadCloser, error) {
	if e, ok := m[path]; ok {
		return io.NopCloser(bytes.NewReader(e.Data)), nil
	}
	return nil, fs.ErrNotExist
}

func (m Memory) Lstat(path string) (fs.FileInfo, error) {
	if e, ok := m[path]; ok {
		return e.FileInfo, nil
	}
	return nil, fs.ErrNotExist
}

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

type Memory map[string]MemoryEntry

type MemoryEntry struct {
	FileInfo fs.FileInfo
	Data     []byte
}

type MemoryFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (fi MemoryFileInfo) Name() string {
	return fi.name
}

func (fi MemoryFileInfo) Size() int64 {
	return fi.size
}

func (fi MemoryFileInfo) Mode() fs.FileMode {
	return fi.mode
}

func (fi MemoryFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi MemoryFileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (fi MemoryFileInfo) Sys() any {
	return nil
}
