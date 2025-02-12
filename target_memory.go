// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	p "path"
	"slices"
	"strings"
	"sync"
	"time"
)

// TargetMemory is an in-memory filesystem implementation that can be used to
// create, read, and write files in memory. It can also be used to create
// directories and symlinks. Permissions (such as owner or group) are not enforced.
type TargetMemory struct {
	fs.FS
	files sync.Map // map[string]*MemoryEntry
}

// NewTargetMemory creates a new in-memory filesystem.
func NewTargetMemory() *TargetMemory {
	return &TargetMemory{
		files: sync.Map{},
	}
}

// CreateFile creates a new file in the in-memory filesystem. The file is created with the given mode.
// If the overwrite flag is set to false and the file already exists, an error is returned. If the overwrite
// flag is set to true, the file is overwritten. The maxSize parameter can be used to limit the size of the file.
// If the file exceeds the maxSize, an error is returned. If the file is created successfully, the number of bytes
// written is returned.
func (m *TargetMemory) CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {
	if !fs.ValidPath(path) {
		return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: fs.ErrInvalid}
	}

	// get real path
	dir, name := p.Split(path)
	dir = p.Clean(dir)
	realDir, err := m.resolvePath(dir)
	if err != nil {
		return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: err}
	}

	// verify that realDir is a directory
	realDirMe, err := m.resolveEntry(realDir)
	if err != nil {
		return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: err}
	}
	if !realDirMe.fileInfo.Mode().IsDir() {
		return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: fs.ErrInvalid}
	}
	realPath := p.Join(realDir, name)

	// get entry
	e, ok := m.files.Load(realPath)
	if !ok {
		return m.createFile(realPath, mode, src, maxSize)
	}
	me := e.(*memoryEntry)

	// handle directory
	if me.fileInfo.Mode().IsDir() {
		return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: fs.ErrExist}
	}

	// remove existing entry and create file
	if overwrite {
		if err := m.Remove(realPath); err != nil {
			return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: err}
		}
		return m.createFile(realPath, mode, src, maxSize)
	}

	// return error if file already exists
	return 0, &fs.PathError{Op: "CreateFile", Path: path, Err: fs.ErrExist}

}

func (m *TargetMemory) createFile(path string, mode fs.FileMode, src io.Reader, maxSize int64) (int64, error) {
	// get name
	name := p.Base(path)

	// create byte buffered writer
	var buf bytes.Buffer
	w := limitWriter(&buf, maxSize)

	// write to buffer
	n, err := io.Copy(w, src)
	if err != nil {
		return n, &fs.PathError{Op: "createFile", Path: path, Err: err}
	}

	// create entry
	m.files.Store(path, &memoryEntry{
		fileInfo: &memoryFileInfo{name: name, size: n, mode: mode.Perm(), accessTime: time.Now(), modTime: time.Now()},
		data:     buf.Bytes(),
		lock:     sync.RWMutex{},
	})
	return n, nil

}

// CreateDir creates a new directory in the in-memory filesystem.
// If the directory already exists, nothing is done. If the directory does not exist, it is created.
// The directory is created with the given mode. If the directory is created successfully, nil is returned.
func (m *TargetMemory) CreateDir(path string, mode fs.FileMode) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "CreateDir", Path: path, Err: fs.ErrInvalid}
	}

	// get real path
	name := p.Base(path)
	dir := p.Dir(path)
	realDir, err := m.resolvePath(dir)
	if err != nil {
		return &fs.PathError{Op: "CreateDir", Path: path, Err: err}
	}

	// verify that realDir is a directory
	realDirMe, err := m.resolveEntry(realDir)
	if err != nil {
		return &fs.PathError{Op: "CreateDir", Path: path, Err: err}
	}
	if !realDirMe.fileInfo.Mode().IsDir() {
		return &fs.PathError{Op: "CreateDir", Path: path, Err: fs.ErrInvalid}
	}
	realPath := p.Join(realDir, name)

	// load entry
	e, ok := m.files.Load(realPath)

	// handle entry
	switch {
	case !ok: // create directory if it does not exist
		m.files.Store(realPath, &memoryEntry{
			fileInfo: &memoryFileInfo{name: name, mode: mode.Perm() | fs.ModeDir},
		})
		return nil

	case e.(*memoryEntry).fileInfo.Mode().IsDir(): // directory already exists
		return nil

	default: // entry exists but is not a directory
		return &fs.PathError{Op: "CreateDir", Path: path, Err: fs.ErrExist}

	}
}

// CreateSymlink creates a new symlink in the in-memory filesystem.
// If the overwrite flag is set to false and the symlink already exists, an error is returned.
// If the overwrite flag is set to true, the symlink is overwritten. If the symlink is created successfully, nil is returned.
func (m *TargetMemory) CreateSymlink(oldName string, newName string, overwrite bool) error {
	if !fs.ValidPath(newName) {
		return &fs.PathError{Op: "CreateSymlink", Path: newName, Err: fs.ErrInvalid}
	}

	// get real path
	dir, name := p.Split(newName)
	dir = p.Clean(dir)
	realDir, err := m.resolvePath(dir)
	if err != nil {
		return &fs.PathError{Op: "CreateSymlink", Path: newName, Err: err}
	}

	// verify that realDir is a directory
	realDirMe, err := m.resolveEntry(realDir)
	if err != nil {
		return &fs.PathError{Op: "CreateSymlink", Path: newName, Err: err}
	}
	if !realDirMe.fileInfo.Mode().IsDir() {
		return &fs.PathError{Op: "CreateSymlink", Path: newName, Err: fs.ErrInvalid}
	}
	realPath := p.Join(realDir, name)

	// resolve real path
	e, ok := m.files.Load(realPath)

	switch {

	// create symlink, bc/it does not exist
	case !ok:
		m.files.Store(realPath, &memoryEntry{
			fileInfo: &memoryFileInfo{name: name, mode: 0777 | fs.ModeSymlink},
			data:     []byte(oldName),
		})
		return nil

	// directories cannot be overwritten
	case e.(*memoryEntry).fileInfo.Mode().IsDir():
		return &fs.PathError{Op: "CreateSymlink", Path: newName, Err: fs.ErrExist}

	// remove existing entry and create symlink
	case overwrite:
		if err := m.Remove(realPath); err != nil {
			return &fs.PathError{Op: "CreateSymlink", Path: realPath, Err: err}
		}
		m.files.Store(realPath, &memoryEntry{
			fileInfo: &memoryFileInfo{name: name, mode: 0777 | fs.ModeSymlink},
			data:     []byte(oldName),
		})
		return nil

	// error, if entry already exists
	default:
		return &fs.PathError{Op: "CreateSymlink", Path: newName, Err: fs.ErrExist}
	}
}

// Open implements the [io/fs.FS] interface. It opens the file at the given path.
// If the file does not exist, an error is returned. If the file is a directory,
// an error is returned. If the file is a symlink, the target of the symlink is returned.
func (m *TargetMemory) Open(path string) (fs.File, error) {
	// traverse the path & symlinks to get to the real path
	actualPath, err := m.resolvePath(path)
	if err != nil {
		return nil, &fs.PathError{Op: "Open", Path: path, Err: err}
	}

	// get entry
	me, err := m.resolveEntry(actualPath)
	if err != nil {
		return nil, &fs.PathError{Op: "Open", Path: path, Err: err}
	}

	// check if it is a directory
	if me.fileInfo.Mode().IsDir() {
		return &dirEntry{memoryEntry: me, memory: m, path: actualPath, readDirCounter: 0}, nil
	}

	// get lock and create reader
	me.lock.RLock()
	return &fileEntry{memoryEntry: me, reader: bytes.NewReader(me.data)}, nil
}

// Chmod changes the mode of the file at the given path. If the file does not exist, an error is returned.
func (m *TargetMemory) Chmod(path string, mode fs.FileMode) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "Chmod", Path: path, Err: fs.ErrInvalid}
	}
	me, err := m.resolveEntry(path)
	if err != nil {
		return &fs.PathError{Op: "Chmod", Path: path, Err: err}
	}
	me.lock.Lock()
	defer me.lock.Unlock()
	// inverse & with 0777 to remove the file mode bits and then or with the new mode bits
	me.fileInfo.(*memoryFileInfo).mode = (me.fileInfo.(*memoryFileInfo).mode &^ 0777) | mode.Perm()
	return nil
}

// Chtime changes the access and modification times of the file at the given path.
// If the file does not exist, an error is returned.
func (m *TargetMemory) Chtimes(path string, atime time.Time, mtime time.Time) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "Chtimes", Path: path, Err: fs.ErrInvalid}
	}
	me, err := m.resolveEntry(path)
	if err != nil {
		return &fs.PathError{Op: "Chtimes", Path: path, Err: err}
	}
	me.lock.Lock()
	defer me.lock.Unlock()
	me.fileInfo.(*memoryFileInfo).accessTime = atime
	me.fileInfo.(*memoryFileInfo).modTime = mtime
	return nil
}

// Chown changes the owner and group of the file at the given path.
// If the file does not exist, an error is returned.
func (m *TargetMemory) Chown(path string, uid, gid int) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "Chtimes", Path: path, Err: fs.ErrInvalid}
	}
	me, err := m.resolveEntry(path)
	if err != nil {
		return &fs.PathError{Op: "Chtimes", Path: path, Err: err}
	}
	me.lock.Lock()
	defer me.lock.Unlock()
	me.fileInfo.(*memoryFileInfo).uid = uid
	me.fileInfo.(*memoryFileInfo).gid = gid
	return nil
}

// Lchtimes changes the access and modification times of the file at the given path.
// If the file does not exist, an error is returned.
func (m *TargetMemory) Lchtimes(path string, atime time.Time, mtime time.Time) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "Lchtimes", Path: path, Err: fs.ErrInvalid}
	}
	me, err := m.resolveEntry(path)
	if err != nil {
		return &fs.PathError{Op: "Lchtimes", Path: path, Err: err}
	}
	me.lock.Lock()
	defer me.lock.Unlock()
	me.fileInfo.(*memoryFileInfo).accessTime = atime
	me.fileInfo.(*memoryFileInfo).modTime = mtime
	return nil
}

type dirEntry struct {
	*memoryEntry
	memory         *TargetMemory
	path           string
	readDirCounter int
	closed         bool
}

// ReadDir implements the [io/fs.ReadDirFile] interface. It reads the directory
// named by the entry and returns a list of directory entries sorted by filename.
// If n > 0, ReadDir returns at most n DirEntry. In this case, if there are fewer
// than n DirEntry, it also returns an io.EOF error.
// If n <= 0, ReadDir returns all DirEntry. In this case, if there are no DirEntry,
// it returns an empty slice. ReadDir may return io.EOF if an error occurred during
// reading the directory.
func (de *dirEntry) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, err := de.memory.ReadDir(de.path)
	if err != nil {
		return nil, &fs.PathError{Op: "ReadDir", Path: de.path, Err: err}
	}

	// return all entries if n <= 0 that have not been read yet
	if n <= 0 {
		if de.readDirCounter >= len(entries) {
			// return empty slice
			return nil, nil
		}
		entries = entries[de.readDirCounter:]
		de.readDirCounter = de.readDirCounter + len(entries)
		return entries, nil
	}

	// check that readDirCounter is not out of bounds
	// if it is, return an EOF error
	if de.readDirCounter >= len(entries) {
		return nil, io.EOF
	}

	// check if n is greater than the number of entries
	// left in the slice. If so, return the remaining entries
	// and an EOF error
	if n >= len(entries[de.readDirCounter:]) {
		entries = entries[de.readDirCounter:]
		de.readDirCounter = de.readDirCounter + len(entries)
		return entries, io.EOF
	}

	de.readDirCounter = de.readDirCounter + n
	return entries[de.readDirCounter-n : de.readDirCounter], nil
}

func (de *dirEntry) Read(p []byte) (int, error) {
	return 0, &fs.PathError{Op: "Read", Path: de.fileInfo.Name(), Err: fmt.Errorf("is a directory")}
}

// Close implements the [io/fs.File] interface.
func (de *dirEntry) Close() error {
	de.closed = true
	return nil
}

// fileEntry is a [io/fs.File] implementation for the in-memory filesystem
type fileEntry struct {
	*memoryEntry
	closed bool
	reader io.Reader
}

// Read implements the [io/fs.File] interface.
func (fe *fileEntry) Read(p []byte) (int, error) {
	if fe.closed {
		return 0, &fs.PathError{Op: "Read", Path: fe.fileInfo.Name(), Err: fs.ErrClosed}
	}
	return fe.reader.Read(p)
}

// Close implements the [io/fs.File] interface.
func (fe *fileEntry) Close() error {
	if fe.closed {
		return &fs.PathError{Op: "Close", Path: fe.fileInfo.Name(), Err: fs.ErrClosed}
	}
	fe.lock.RUnlock()
	fe.closed = true
	return nil
}

// resolveEntry resolves the entry at the given path. If the path does not exist, an error is returned.
func (m *TargetMemory) resolveEntry(path string) (*memoryEntry, error) {

	// split path and traverse
	name := p.Base(path)
	dir := p.Dir(path)
	existingEntry, err := m.resolvePath(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err != nil {
		return nil, &fs.PathError{Op: "resolveEntry", Path: path, Err: err}
	}
	realPath := p.Join(existingEntry, name)

	// handle root directory
	if realPath == "." {
		me := &memoryEntry{
			fileInfo: &memoryFileInfo{name: ".", mode: 0777 ^ fs.ModeDir},
		}
		return me, nil
	}

	// get entry
	e, ok := m.files.Load(realPath)
	if !ok {
		return nil, fs.ErrNotExist
	}

	// return entry
	return e.(*memoryEntry), nil
}

// resolvePath resolves the path and traverses symlinks. If anything went
// wrong or the paths are in a symlink loop or the path is invalid, an error
// is returned. If the path is empty, the current directory is returned.
func (m *TargetMemory) resolvePath(path string) (string, error) {
	// handle empty path
	if path == "" {
		return ".", nil
	}

	// ensure path is valid
	if !fs.ValidPath(path) {
		return "", &fs.PathError{Op: "resolvePath", Path: path, Err: fs.ErrInvalid}
	}

	// split path and traverse
	resultingPath := ""
	parts := strings.Split(path, "/")
	for i, part := range parts {
		resultingPath = p.Clean(p.Join(resultingPath, part))

		// traverse symlinks, protect against endless loops
		for j := 255; j >= 0; j-- {

			if j == 0 {
				return "", &fs.PathError{Op: "resolvePath", Path: path, Err: fs.ErrInvalid}
			}

			// check if resulting path is valid
			if !fs.ValidPath(resultingPath) {
				return "", &fs.PathError{Op: "resolvePath", Path: path, Err: fs.ErrInvalid}
			}

			// check if resulting path is root
			if resultingPath == "." {
				break
			}

			// check if resulting path does exist
			e, ok := m.files.Load(resultingPath)
			if !ok {
				return "", fs.ErrNotExist
			}
			me := e.(*memoryEntry)

			// check if we are in a directory, if so break the loop
			// and continue with the next part of the path
			if me.fileInfo.Mode().IsDir() {
				break
			}

			// check if we are pointing to a file, if so check if we are
			// at the end of the path
			if me.fileInfo.Mode().IsRegular() {
				if i < len(parts)-1 {
					return resultingPath, &fs.PathError{Op: "resolvePath", Path: path, Err: fs.ErrInvalid}
				}
				break
			}

			// check if we are in a symlink, if so resolve the symlink
			// and repeat the previous checks with the resolved path
			if me.fileInfo.Mode()&fs.ModeSymlink != 0 {
				linkTarget := p.Join(p.Dir(resultingPath), string(me.data))
				resultingPath = linkTarget
			}
		}
	}

	return resultingPath, nil

}

// Lstat returns the FileInfo for the given path. If the path is a symlink, the FileInfo for the symlink is returned.
// If the path does not exist, an error is returned.
//
// golang/go#49580 proposes adding a standard io/fs.SymlinkFS interface to the io/fs package, which discusses
// if the Lstat method should be moved to the io/fs.SymlinkFS interface.
// ref: https://github.com/golang/go/issues/49580#issuecomment-2344253737
func (m *TargetMemory) Lstat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "Lstat", Path: path, Err: fs.ErrInvalid}
	}

	// check if path exist, traverse symlinks in the path to get to the real
	// path
	me, err := m.resolveEntry(path)
	if err != nil {
		return nil, &fs.PathError{Op: "Lstat", Path: path, Err: err}
	}

	// return file info copy
	mfi := me.fileInfo.(*memoryFileInfo)
	return &memoryFileInfo{
		name:       mfi.Name(),
		size:       mfi.Size(),
		mode:       mfi.Mode(),
		accessTime: mfi.AccessTime(),
		modTime:    mfi.ModTime(),
		gid:        mfi.Gid(),
		uid:        mfi.Uid(),
	}, nil
}

// Stat implements the [io/fs.File] and [io/fs.StatFS] interfaces. It returns the
// [io/fs.FileInfo] for the given path.
func (m *TargetMemory) Stat(path string) (fs.FileInfo, error) {
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "Stat", Path: path, Err: fs.ErrInvalid}
	}

	// check if path exist, traverse symlinks in the path to get to the real
	// file
	actualPath, err := m.resolvePath(path)
	if err != nil {
		return nil, &fs.PathError{Op: "Stat", Path: path, Err: err}
	}

	// get entry
	me, err := m.resolveEntry(actualPath)
	if err != nil {
		return nil, &fs.PathError{Op: "Stat", Path: path, Err: err}
	}

	// return file info copy
	return &memoryFileInfo{
		name:    me.fileInfo.Name(),
		size:    me.fileInfo.Size(),
		mode:    me.fileInfo.Mode(),
		modTime: me.fileInfo.ModTime(),
	}, nil
}

// Readlink returns the target of the symlink at the given path. If the
// path is not a symlink, an error is returned.
//
// golang/go#49580 proposes adding a standard io/fs.SymlinkFS interface
// to the io/fs package. If this proposal is accepted, the Readlink
// method will be moved to the io/fs.SymlinkFS interface.
// Until then, the Readlink method is kept not exposed.
func (m *TargetMemory) Readlink(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", &fs.PathError{Op: "Readlink", Path: path, Err: fs.ErrInvalid}
	}

	// get entry
	me, err := m.resolveEntry(path)
	if err != nil {
		return "", &fs.PathError{Op: "Readlink", Path: path, Err: err}
	}

	// handle symlink
	if me.fileInfo.Mode()&fs.ModeSymlink != 0 {
		return string(me.data), nil
	}

	return "", &fs.PathError{Op: "Readlink", Path: path, Err: fs.ErrInvalid}
}

// Remove removes the file or directory at the given path. If the path
// is invalid, an error is returned. If the path does not exist, no error
// is returned.
func (m *TargetMemory) Remove(path string) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "Remove", Path: path, Err: fs.ErrInvalid}
	}

	// delete sub-entries if path is a directory
	e, ok := m.files.Load(path)
	if !ok {
		return nil
	}
	me := e.(*memoryEntry)
	if me.fileInfo.Mode().IsDir() {
		entries, err := m.ReadDir(path)
		if err != nil {
			return &fs.PathError{Op: "Remove", Path: path, Err: err}
		}
		for _, entry := range entries {
			if err := m.Remove(p.Join(path, entry.Name())); err != nil {
				return &fs.PathError{Op: "Remove", Path: path, Err: err}
			}
		}
	}

	// delete entry
	if !me.lock.TryLock() {
		return &fs.PathError{Op: "Remove", Path: path, Err: fmt.Errorf("file is in use")}
	}
	defer me.lock.Unlock()

	// delete entry
	m.files.Delete(path)
	return nil
}

// ReadDir implements the [io/fs.ReadDirFS] interface. It reads
// the directory named by dirname and returns a list of directory
// entries sorted by filename.
func (m *TargetMemory) ReadDir(path string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "ReadDir", Path: path, Err: fs.ErrInvalid}
	}

	// get real path
	realPath, err := m.resolvePath(path)
	if err != nil {
		return nil, &fs.PathError{Op: "ReadDir", Path: path, Err: err}
	}

	// get entry and check if it is a directory
	if e, err := m.resolveEntry(realPath); err != nil {
		return nil, &fs.PathError{Op: "ReadDir", Path: path, Err: err}
	} else if !e.fileInfo.Mode().IsDir() {
		return nil, &fs.PathError{Op: "ReadDir", Path: path, Err: fs.ErrInvalid}
	}

	// get all entries in the directory
	var entries []fs.DirEntry
	m.files.Range(func(path, entry any) bool {
		if p.Dir(path.(string)) == realPath {
			entries = append(entries, entry.(*memoryEntry))
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
func (m *TargetMemory) ReadFile(path string) ([]byte, error) {
	// open file for reading to ensure that symlinks are resolved
	f, err := m.Open(path)
	if err != nil {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: fs.ErrNotExist}
	}
	defer f.Close()

	// check if it is a directory
	stat, err := f.Stat()
	if err != nil {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: err}
	}
	if stat.IsDir() {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: fs.ErrInvalid}
	}

	// read file data
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: err}
	}

	return data, nil
}

// Sub implements the [io/fs.SubFS] interface. It returns a
// new FS representing the subtree rooted at dir.
func (m *TargetMemory) Sub(subPath string) (fs.FS, error) {
	// get real path
	realPath, err := m.resolvePath(subPath)
	if err != nil {
		return nil, &fs.PathError{Op: "Sub", Path: subPath, Err: err}
	}

	// handle root directory
	if realPath == "." {
		return m, nil
	}

	// load entry from map
	me, err := m.resolveEntry(realPath)
	if err != nil {
		return nil, &fs.PathError{Op: "Sub", Path: subPath, Err: err}
	}

	// check if it is not a directory
	if !me.fileInfo.Mode().IsDir() {
		return nil, &fs.PathError{Op: "Sub", Path: subPath, Err: fs.ErrInvalid}
	}

	// handle directories
	realPath = p.Clean(realPath) + "/"
	tm := NewTargetMemory()
	m.files.Range(func(key, entry any) bool {
		path := key.(string)
		if strings.HasPrefix(path, realPath) {
			tm.files.Store(path[len(realPath):], entry)
		}
		return true
	})

	return tm, nil
}

// Glob implements the [io/fs.GlobFS] interface. It returns
// the names of all files matching pattern.
func (m *TargetMemory) Glob(pattern string) ([]string, error) {
	if !fs.ValidPath(pattern) {
		return nil, &fs.PathError{Op: "Glob", Path: pattern, Err: fs.ErrInvalid}
	}

	var matches []string
	var err error
	m.files.Range(func(key, entry any) bool {
		path := key.(string)
		match, matchErr := p.Match(pattern, path)
		if matchErr != nil {
			err = &fs.PathError{Op: "Glob", Path: pattern, Err: matchErr}
			return false
		}
		if match {
			matches = append(matches, path)
		}
		return true
	})

	// sort matches
	slices.Sort(matches)

	return matches, err
}

// memoryEntry is a File implementation for the in-memory filesystem
type memoryEntry struct {
	fileInfo fs.FileInfo
	data     []byte
	lock     sync.RWMutex
}

// Stat implements the [io/fs.File] interface.
func (me *memoryEntry) Stat() (fs.FileInfo, error) {
	return me.fileInfo, nil
}

// Name implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) Name() string {
	return me.fileInfo.Name()
}

// IsDir implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) IsDir() bool {
	return me.fileInfo.IsDir()
}

// Type implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) Type() fs.FileMode {
	return me.fileInfo.Mode().Type()
}

// Info implements the [io/fs.DirEntry] interface.
func (me *memoryEntry) Info() (fs.FileInfo, error) {
	return me.fileInfo, nil
}

// memoryFileInfo is a FileInfo implementation for the in-memory filesystem
type memoryFileInfo struct {
	accessTime time.Time
	gid        int
	name       string
	mode       fs.FileMode
	modTime    time.Time
	size       int64
	uid        int
}

// AccessTime returns the access time of the file
func (fi *memoryFileInfo) AccessTime() time.Time {
	return fi.accessTime
}

// Gid returns the group id of the file
func (fi *memoryFileInfo) Gid() int {
	return fi.gid
}

// IsDir implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

// Mode implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) Mode() fs.FileMode {
	return fi.mode
}

// ModTime implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) ModTime() time.Time {
	return fi.modTime
}

// Name implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) Name() string {
	return fi.name
}

// Size implements [io/fs.FileInfo] interface
func (fi *memoryFileInfo) Size() int64 {
	return fi.size
}

// Sys implements [io/fs.FileInfo] interface, but returns always nil
func (fi *memoryFileInfo) Sys() any {
	return nil
}

// Uid returns the user id of the file
func (fi *memoryFileInfo) Uid() int {
	return fi.uid
}
