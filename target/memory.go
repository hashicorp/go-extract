package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	p "path"
	"path/filepath"
	"slices"
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
		return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: fs.ErrInvalid}
	}
	if !overwrite {
		if _, ok := m.files.Load(path); ok {
			return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: fs.ErrExist}
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
		return &fs.PathError{Op: "CreateDir", Path: path, Err: fs.ErrInvalid}
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
		return &fs.PathError{Op: "CreateSymlink", Path: newName, Err: fs.ErrInvalid}
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
		return nil, &fs.PathError{Op: "Open", Path: path, Err: fs.ErrInvalid}
	}

	// get entry
	e, ok := m.files.Load(path)

	// file does not exist
	if !ok {
		return nil, &fs.PathError{Op: "Open", Path: path, Err: fs.ErrNotExist}
	}

	// handle directory
	me := e.(*memoryEntry)
	if me.FileInfo.Mode()&fs.ModeDir != 0 {
		return nil, &fs.PathError{Op: "Open", Path: path, Err: fs.ErrInvalid}
	}

	// handle symlink
	if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
		linkTarget := resolveLink(path, me.Data)
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

// resolveLink resolves the target of a symlink. The target is resolved by joining the
// directory of the symlink with the target name. The target name is read from the symlink data.
func resolveLink(path string, data []byte) string {
	linkTarget := string(data)
	linkDir := p.Dir(path)
	p.Join(linkDir, linkTarget)
	return linkTarget
}

// Lstat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the symlink is returned.
// If the path does not exist, an error is returned.
func (m *Memory) Lstat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "Lstat", Path: path, Err: fs.ErrInvalid}
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*memoryEntry)
		return me.FileInfo, nil
	}
	return nil, &fs.PathError{Op: "Lstat", Path: path, Err: fs.ErrNotExist}
}

// Stat implements the [io/fs.StatFS] interface. It returns the
// FileInfo for the given path.
func (m *Memory) Stat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "Stat", Path: path, Err: fs.ErrInvalid}
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*memoryEntry)
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			linkTarget := resolveLink(path, me.Data)
			return m.Stat(linkTarget)
		}
		return me.FileInfo, nil
	}
	return nil, &fs.PathError{Op: "Stat", Path: path, Err: fs.ErrNotExist}
}

// Readlink returns the target of the symlink at the given path. If the
// path is not a symlink, an error is returned.
func (m *Memory) Readlink(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", &fs.PathError{Op: "Readlink", Path: path, Err: fs.ErrInvalid}
	}
	if e, ok := m.files.Load(path); ok {
		me := e.(*memoryEntry)
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			return string(me.Data), nil
		}
		return "", &fs.PathError{Op: "Readlink", Path: path, Err: fs.ErrInvalid}
	}
	return "", &fs.PathError{Op: "Readlink", Path: path, Err: fs.ErrNotExist}
}

// Remove removes the file or directory at the given path. If the path
// is invalid, an error is returned. If the path does not exist, no error
// is returned.
func (m *Memory) Remove(path string) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "Remove", Path: path, Err: fs.ErrInvalid}
	}
	m.files.Delete(path)
	return nil
}

// ReadDir implements the [io/fs.ReadDirFS] interface. It reads
// the directory named by dirname and returns a list of directory
// entries sorted by filename.
func (m *Memory) ReadDir(path string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "ReadDir", Path: path, Err: fs.ErrInvalid}
	}

	// handle non-root directory
	if path != "." {

		// load entry from map
		e, ok := m.files.Load(path)
		if !ok {
			return nil, &fs.PathError{Op: "ReadDir", Path: path, Err: fs.ErrNotExist}
		}
		me := e.(*memoryEntry)

		// handle symlink
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			linkTarget := resolveLink(path, me.Data)
			return m.ReadDir(linkTarget)
		}

		// handle file
		if me.FileInfo.Mode().IsRegular() {
			return nil, &fs.PathError{Op: "ReadDir", Path: path, Err: fs.ErrInvalid}
		}

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
	slices.SortStableFunc(entries, func(i, j fs.DirEntry) int {
		return strings.Compare(i.Name(), j.Name())
	})

	return entries, nil
}

// ReadFile implements the [io/fs.ReadFileFS] interface. It
// reads the file named by filename and returns the contents.
func (m *Memory) ReadFile(path string) ([]byte, error) {
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: fs.ErrInvalid}
	}

	// open file for reading to ensure that symlinks are resolved
	f, err := m.Open(path)
	if err != nil {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: fs.ErrNotExist}
	}
	defer f.Close()

	// read file data
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: err}
	}

	return data, nil
}

// Sub implements the [io/fs.SubFS] interface. It returns a
// new FS representing the subtree rooted at dir.
func (m *Memory) Sub(subPath string) (fs.FS, error) {
	if !fs.ValidPath(subPath) {
		return nil, &fs.PathError{Op: "Sub", Path: subPath, Err: fs.ErrInvalid}
	}

	// handle non-root directory
	if subPath != "." {

		// load entry from map
		e, ok := m.files.Load(subPath)
		if !ok {
			return nil, &fs.PathError{Op: "Sub", Path: subPath, Err: fs.ErrNotExist}
		}
		me := e.(*memoryEntry)

		// handle symlink
		if me.FileInfo.Mode()&fs.ModeSymlink != 0 {
			linkTarget := resolveLink(subPath, me.Data)
			return m.Sub(linkTarget)
		}

		// handle files
		if me.FileInfo.Mode().IsRegular() {
			return nil, &fs.PathError{Op: "Sub", Path: subPath, Err: fs.ErrInvalid}
		}
	}

	// handle directories
	subPath = p.Clean(subPath) + "/"
	subFS := NewMemory()
	m.files.Range(func(dir, entry any) bool {
		if strings.HasPrefix(dir.(string), subPath) {
			subFS.files.Store(dir.(string)[len(subPath):], entry)
		}
		return true
	})

	return subFS, nil
}

// Glob implements the [io/fs.GlobFS] interface. It returns
// the names of all files matching pattern.
func (m *Memory) Glob(pattern string) ([]string, error) {
	if !fs.ValidPath(pattern) {
		return nil, &fs.PathError{Op: "Glob", Path: pattern, Err: fs.ErrInvalid}
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
