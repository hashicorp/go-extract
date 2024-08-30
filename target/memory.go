package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Memory is an in-memory filesystem implementation. It is a map of file paths to MemoryEntry.
// The MemoryEntry contains the file information and the file data.
// The Memory filesystem can be used to create, read, and write files in memory. It can also be
// used to create directories and symlinks. Permissions on entries (owner, group, others) are
// not enforced. Entries can be accessed by the path as a key in the map, or by calling
// the m.Open(<path>) function.
type Memory struct {
	files sync.Map // map[string]*MemoryEntry
}

// NewMemory creates a new in-memory filesystem.
func NewMemory() Target {
	return &Memory{
		files: sync.Map{},
	}
}

// CreateFile creates a new file in the in-memory filesystem. The file is created with the given mode.
// If the overwrite flag is set to false and the file already exists, an error is returned. If the overwrite
// flag is set to true, the file is overwritten. The maxSize parameter can be used to limit the size of the file.
// If the file exceeds the maxSize, an error is returned. If the file is created successfully, the number of bytes
// written is returned.
func (m *Memory) CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {
	if !fs.ValidPath(path) {
		return 0, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if !overwrite {
		if _, ok := m.files.Load(path); ok {
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
	fName := filepath.Base(path)
	m.files.Store(path, &MemoryEntry{
		FileInfo: &MemoryFileInfo{name: fName, size: n, mode: mode.Perm(), modTime: time.Now()},
		Data:     buf.Bytes(),
	})

	// return number of bytes written
	return n, nil
}

// CreateDir creates a new directory in the in-memory filesystem.
// If the directory already exists, nothing is done. If the directory does not exist, it is created.
// The directory is created with the given mode. If the directory is created successfully, nil is returned.
func (m *Memory) CreateDir(path string, mode fs.FileMode) error {
	if !fs.ValidPath(path) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}

	// check if an entry already exists
	if _, ok := m.files.Load(path); ok {
		return nil
	}

	// create entry
	dName := filepath.Base(path)
	m.files.Store(path, &MemoryEntry{
		FileInfo: &MemoryFileInfo{name: dName, mode: mode.Perm() | fs.ModeDir},
	})

	return nil
}

// CreateSymlink creates a new symlink in the in-memory filesystem.
// If the overwrite flag is set to false and the symlink already exists, an error is returned.
// If the overwrite flag is set to true, the symlink is overwritten. If the symlink is created successfully, nil is returned.
func (m *Memory) CreateSymlink(oldName string, newName string, overwrite bool) error {
	if !fs.ValidPath(newName) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, newName)
	}
	if !overwrite {
		if _, ok := m.files.Load(newName); ok {
			return fmt.Errorf("%w: %s", fs.ErrExist, newName)
		}
	}

	lName := filepath.Base(newName)
	m.files.Store(newName, &MemoryEntry{
		FileInfo: &MemoryFileInfo{name: lName, mode: 0777 | fs.ModeSymlink},
		Data:     []byte(oldName),
	})

	return nil
}

// Open opens the named file for reading. If successful, the file is returned
// as an [io.ReadCloser] which can be used to read the file contents. If the
// file is  a symlink, the target of the symlink is opened. If the file does not
// exist, or is a directory, an error is returned.
func (m *Memory) Open(path string) (fs.File, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}

	// get entry
	e, ok := m.files.Load(path)

	// file does not exist
	if !ok {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
	}

	// handle directory
	me := e.(*MemoryEntry)
	if me.FileInfo.Mode()&fs.ModeDir != 0 {
		return nil, fmt.Errorf("cannot open directory")
	}

	// handle symlink
	if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
		linkTarget := string(me.Data)
		linkTarget = filepath.Join(filepath.Dir(path), linkTarget)
		return m.Open(linkTarget)
	}

	// create copy of entry
	me = &MemoryEntry{
		FileInfo: me.FileInfo,
		Data:     me.Data,
	}

	// return file data
	return me, nil

}

// Lstat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the symlink is returned.
// If the path does not exist, an error is returned.
func (m *Memory) Lstat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*MemoryEntry)
		return me.FileInfo, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Stat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the target of the symlink is returned.
// If the path does not exist, an error is returned.
func (m *Memory) Stat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*MemoryEntry)
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			linkTarget := string(me.Data)
			linkTarget = filepath.Join(filepath.Dir(path), linkTarget)
			return m.Stat(linkTarget)
		}
		return me.FileInfo, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Readlink returns the target of the symlink at the given path. If the path is not a symlink, an error is returned.
// If the path does not exist, an error is returned. If the symlink exists, the target of the symlink is returned.
func (m *Memory) Readlink(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*MemoryEntry)
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			return string(me.Data), nil
		}
		return "", fmt.Errorf("not a symlink: %w: %s", fs.ErrInvalid, path)
	}
	return "", fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Remove removes the entry at the given path. If the path does not exist, an error is returned.
func (m *Memory) Remove(path string) error {
	if !fs.ValidPath(path) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	m.files.Delete(path)
	return nil
}

func (m *Memory) ReadDir(path string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s	", fs.ErrInvalid, path)
	}

	// get all entries in the directory
	var entries []fs.DirEntry
	m.files.Range(func(entryPath, me any) bool {
		if filepath.Dir(entryPath.(string)) == path {
			entries = append(entries, me.(*MemoryEntry))
		}
		return true
	})

	// sort slice of entries based on name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

func (m *Memory) ReadFile(path string) ([]byte, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*MemoryEntry)
		if me.FileInfo.Mode()&fs.ModeDir != 0 {
			return nil, fmt.Errorf("cannot read directory")
		}
		return me.Data, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (m *Memory) Sub(dir string) (fs.FS, error) {
	if !fs.ValidPath(dir) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, dir)
	}

	// Create a new Memory filesystem for the subdirectory
	dir = filepath.Clean(dir) + "/"
	subFS := NewMemory().(*Memory)
	m.files.Range(func(path, entry any) bool {
		if strings.HasPrefix(path.(string), dir) {
			subFS.files.Store(path.(string)[len(dir):], entry)
		}
		return true
	})

	return subFS, nil
}

// Glob returns the names of all files matching pattern or nil if there is no matching file.
func (m *Memory) Glob(pattern string) ([]string, error) {
	if !fs.ValidPath(pattern) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, pattern)
	}

	var matches []string
	m.files.Range(func(path, entry any) bool {
		match, err := filepath.Match(pattern, path.(string))
		if err != nil {
			return false
		}
		if match {
			matches = append(matches, path.(string))
		}
		return true
	})

	return matches, nil
}

// MemoryEntry is an entry in the in-memory filesystem
type MemoryEntry struct {
	FileInfo fs.FileInfo
	Data     []byte
}

func (me *MemoryEntry) Name() string {
	return me.FileInfo.Name()
}

func (me *MemoryEntry) Stat() (fs.FileInfo, error) {
	return me.FileInfo, nil
}

func (me *MemoryEntry) Read(p []byte) (int, error) {
	n := copy(p, me.Data)
	if n == 0 {
		return 0, io.EOF
	}
	me.Data = me.Data[n:]
	return n, nil
}

func (me *MemoryEntry) Close() error {
	return nil
}

func (me *MemoryEntry) IsDir() bool {
	return me.FileInfo.IsDir()
}

func (me *MemoryEntry) Type() fs.FileMode {
	return me.FileInfo.Mode().Type()
}

func (me *MemoryEntry) Info() (fs.FileInfo, error) {
	return me.FileInfo, nil
}

// MemoryFileInfo is a FileInfo implementation for the in-memory filesystem
type MemoryFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

// Name returns the name of the file
func (fi *MemoryFileInfo) Name() string {
	return fi.name
}

// Size returns the size of the file
func (fi *MemoryFileInfo) Size() int64 {
	return fi.size
}

// Mode returns the mode of the file
func (fi *MemoryFileInfo) Mode() fs.FileMode {
	return fi.mode
}

// ModTime returns the modification time of the file
func (fi *MemoryFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir returns true if the file is a directory
func (fi *MemoryFileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

// Sys returns the underlying data source (nil for in-memory filesystem)
func (fi *MemoryFileInfo) Sys() any {
	return nil
}
