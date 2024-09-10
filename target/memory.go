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

// PathSeparator is the path separator used in the in-memory filesystem
const (
	PathSeparator = "/"
)

// NewMemory creates a new in-memory filesystem.
func NewMemory() *Memory {
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
	m.files.Store(path, &memoryEntry{
		FileInfo: &memoryFileInfo{name: fName, size: n, mode: mode.Perm(), modTime: time.Now()},
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
	m.files.Store(path, &memoryEntry{
		FileInfo: &memoryFileInfo{name: dName, mode: mode.Perm() | fs.ModeDir},
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
	m.files.Store(newName, &memoryEntry{
		FileInfo: &memoryFileInfo{name: lName, mode: 0777 | fs.ModeSymlink},
		Data:     []byte(oldName),
	})

	return nil
}

// Open implements the [io/fs.FS] interface. It opens the file at the given path.
// If the file does not exist, an error is returned. If the file is a directory,
// an error is returned. If the file is a symlink, the target of the symlink is returned.
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
	me := e.(*memoryEntry)
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
	me = &memoryEntry{
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
		me := e.(*memoryEntry)
		return me.FileInfo, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Stat implements the [io/fs.StatFS] interface. It returns the
// FileInfo for the given path.
func (m *Memory) Stat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*memoryEntry)
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			linkTarget := string(me.Data)
			linkTarget = filepath.Join(filepath.Dir(path), linkTarget)
			return m.Stat(linkTarget)
		}
		return me.FileInfo, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Readlink returns the target of the symlink at the given path. If the
// path is not a symlink, an error is returned.
func (m *Memory) Readlink(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*memoryEntry)
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			return string(me.Data), nil
		}
		return "", fmt.Errorf("not a symlink: %w: %s", fs.ErrInvalid, path)
	}
	return "", fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Remove removes the file or directory at the given path. If the path
// is invalid, an error is returned. If the path does not exist, no error
// is returned.
func (m *Memory) Remove(path string) error {
	if !fs.ValidPath(path) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	m.files.Delete(path)
	return nil
}

// ReadDir implements the [io/fs.ReadDirFS] interface. It reads
// the directory named by dirname and returns a list of
func (m *Memory) ReadDir(path string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s	", fs.ErrInvalid, path)
	}

	// get all entries in the directory
	var entries []fs.DirEntry
	m.files.Range(func(entryPath, me any) bool {
		if filepath.Dir(entryPath.(string)) == path {
			entries = append(entries, me.(*memoryEntry))
		}
		return true
	})

	// sort slice of entries based on name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// ReadFile implements the [io/fs.ReadFileFS] interface. It
// reads the file named by filename and returns the contents.
func (m *Memory) ReadFile(path string) ([]byte, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, path)
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*memoryEntry)
		if me.FileInfo.Mode()&fs.ModeDir != 0 {
			return nil, fmt.Errorf("cannot read directory")
		}
		return me.Data, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// Sub implements the [io/fs.SubFS] interface. It returns a
// new FS representing the subtree rooted at dir.
func (m *Memory) Sub(dir string) (fs.FS, error) {
	if !fs.ValidPath(dir) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, dir)
	}

	// Create a new Memory filesystem for the subdirectory
	dir = filepath.Clean(dir) + "/"
	subFS := NewMemory()
	m.files.Range(func(path, entry any) bool {
		if strings.HasPrefix(path.(string), dir) {
			subFS.files.Store(path.(string)[len(dir):], entry)
		}
		return true
	})

	return subFS, nil
}

// Glob implements the [io/fs.Glob] interface. It returns
// the names of all files matching pattern.
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

// memoryEntry is a File implementation for the in-memory filesystem
type memoryEntry struct {
	FileInfo fs.FileInfo
	Data     []byte
}

// Stat implements the [io/fs.File] interface.
func (me *memoryEntry) Stat() (fs.FileInfo, error) {
	return me.FileInfo, nil
}

// Read implements the [io/fs.File] interface.
func (me *memoryEntry) Read(p []byte) (int, error) {
	n := copy(p, me.Data)
	if n == 0 {
		return 0, io.EOF
	}
	me.Data = me.Data[n:]
	return n, nil
}

// Close implements the [io/fs.File] interface.
func (me *memoryEntry) Close() error {
	return nil
}

// Name implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) Name() string {
	return me.FileInfo.Name()
}

// IsDir implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) IsDir() bool {
	return me.FileInfo.IsDir()
}

// Type implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) Type() fs.FileMode {
	return me.FileInfo.Mode().Type()
}

// Info implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) Info() (fs.FileInfo, error) {
	return me.FileInfo, nil
}

// memoryFileInfo is a FileInfo implementation for the in-memory filesystem
type memoryFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

// Name implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) Name() string {
	return fi.name
}

// Size implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) Size() int64 {
	return fi.size
}

// Mode implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) Mode() fs.FileMode {
	return fi.mode
}

// ModTime implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

// Sys implements [io/fs.FileInfo] interface, but returns always nil
func (fi *memoryFileInfo) Sys() any {
	return nil
}
