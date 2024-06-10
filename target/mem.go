package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Name    string
	Mode    fs.FileMode
	Content []byte
}

// Noop is a target that does nothing.
type Noop struct {
	Entries map[string]Entry
}

// NewMemTarget returns a new Noop target.
func NewMemTarget() *Noop {
	return &Noop{
		Entries: map[string]Entry{},
	}
}

// CreateDir does nothing.
func (n *Noop) CreateDir(path string, mode fs.FileMode) error {

	// check empty path
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// get parent
	parent := filepath.Dir(path)

	// check if parent is different from path, if not it is root
	if parent == path {
		n.Entries[parent] = Entry{
			Name: parent,
			Mode: fs.ModeDir | mode.Perm(),
		}
		return nil
	}

	// check if parent exists
	if _, ok := n.Entries[parent]; !ok {
		if err := n.CreateDir(parent, mode); err != nil {
			return fmt.Errorf("failed to create parent directory: %s", err)
		}
	}

	// check if parent is writable
	if !n.Entries[parent].Mode.IsDir() {
		return fmt.Errorf("parent is not a directory: %s", parent)
	}

	// check mode of parent
	if n.Entries[parent].Mode.Perm()&0200 == 0 {
		return fmt.Errorf("parent is not writable (%s): %o", parent, n.Entries[parent].Mode.Perm())
	}

	// check if already exists
	if e, ok := n.Entries[path]; ok {
		if e.Mode.IsDir() {
			return nil
		}
		return os.ErrExist
	}

	n.Entries[path] = Entry{
		Name: path,
		Mode: fs.ModeDir | mode,
	}
	return nil
}

// CreateSafeFi	le does nothing.
func (n *Noop) CreateFile(path string, src io.Reader, mode fs.FileMode, overwrite bool, maxSize int64) (int64, error) {
	if _, ok := n.Entries[path]; ok && !overwrite {
		return 0, os.ErrExist
	}

	// check if parent exists, is a directory and is writable
	parent := filepath.Dir(path)
	if parent != "." {
		if _, ok := n.Entries[parent]; !ok {
			return 0, fmt.Errorf("parent directory does not exist: %s", parent)
		}
		if !n.Entries[parent].Mode.IsDir() {
			return 0, fmt.Errorf("parent is not a directory: %s", parent)
		}
		if n.Entries[parent].Mode.Perm()&0200 == 0 {
			return 0, fmt.Errorf("parent is not writable: %s", parent)
		}
	}

	// create dynamic buffer
	buf := new(bytes.Buffer)
	w := limitWriter(buf, maxSize)
	written, err := io.Copy(w, src)

	if err != nil {
		return written, fmt.Errorf("failed to write file: %s", err)
	}

	// store file in map
	n.Entries[path] = Entry{
		Name:    path,
		Mode:    mode,
		Content: buf.Bytes(),
	}

	return written, err
}

// CreateSafeSymlink does nothing.
func (n *Noop) CreateSymlink(oldname string, newname string, overwrite bool) error {
	if _, ok := n.Entries[newname]; ok && !overwrite {
		return os.ErrExist
	}

	n.Entries[newname] = Entry{
		Name:    newname,
		Mode:    fs.ModeSymlink,
		Content: []byte(oldname),
	}
	return nil
}

// Lstat does nothing.
func (n *Noop) Lstat(path string) (fs.FileInfo, error) {

	if path == "." {
		return &fileInfo{
			entry: Entry{
				Name: ".",
				Mode: fs.ModeDir & 0755,
			},
		}, nil
	}

	if _, ok := n.Entries[path]; !ok {
		return nil, os.ErrNotExist
	}
	// convert entry to fs.FileMode
	fi := &fileInfo{
		entry: n.Entries[path],
	}
	return fi, nil
}

type fileInfo struct {
	entry Entry
}

func (fi *fileInfo) Name() string       { return fi.entry.Name }
func (fi *fileInfo) Size() int64        { return int64(len(fi.entry.Content)) } // or provide a real size
func (fi *fileInfo) Mode() fs.FileMode  { return fi.entry.Mode }
func (fi *fileInfo) ModTime() time.Time { return time.Time{} } // or provide a real modification time
func (fi *fileInfo) IsDir() bool        { return fi.entry.Mode.IsDir() }
func (fi *fileInfo) Sys() interface{}   { return nil }
