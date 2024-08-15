package target

import (
	"os"
	"time"

	"github.com/hashicorp/go-extract/config"
)

func TargetFileInfoFromExisting(c config.Config, fi os.FileInfo) TargetFileInfo {
	return NewTargetFileInfo(fi.Name(), fi.Size(), fi.Mode(), fi.ModTime(), fi.IsDir())
}

func NewTargetFileInfo(name string, size int64, mode os.FileMode, modTime time.Time, isDir bool) TargetFileInfo {
	return TargetFileInfo{name: name, size: size, mode: mode, modTime: modTime, isDir: isDir}
}

// TargetFileInfo is a custom implementation of the FileInfo interface.
type TargetFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	sys     interface{}
}

func (fi TargetFileInfo) Name() string       { return fi.name }
func (fi TargetFileInfo) Size() int64        { return fi.size }
func (fi TargetFileInfo) Mode() os.FileMode  { return fi.mode }
func (fi TargetFileInfo) ModTime() time.Time { return fi.modTime }
func (fi TargetFileInfo) IsDir() bool        { return fi.isDir }
func (fi TargetFileInfo) Sys() interface{}   { return fi.sys }
