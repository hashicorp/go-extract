// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	p "path"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	extract "github.com/hashicorp/go-extract"
)

func TestMemoryOpen(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPerm := 0644
	testNotExist := "notexist"
	testDir := "dir"
	testLink := "link"

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// open the file
	f, err := tm.Open(testPath)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}
	defer f.Close()

	// check the file permissions
	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	if int(stat.Mode()&fs.ModePerm) != int(testPerm) {
		t.Fatalf("Open() failed: expected %d, got %d", testPerm, stat.Mode().Perm())
	}

	// read the file
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() failed: %s", err)
	}

	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("ReadAll() failed: expected %s, got %s", testContent, data)
	}

	// open a file that does not exist
	if _, err := tm.Open(testNotExist); err == nil {
		t.Fatalf("Open() failed: expected error, got nil")
	}

	// create a directory
	if err := tm.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("failed to perform CreateDir(): %s", err)
	}

	// open the directory
	if _, err := tm.Open(testDir); err != nil {
		t.Fatalf("failed to Open() directory: %s", err)
	}

	// create a symlink
	if err := tm.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// open the symlink
	f, err = tm.Open(testLink)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}
	defer f.Close()

	// read content of the symlink
	data, err = io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() on symlink failed: %s", err)
	}

	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("ReadAll() on symlink failed: expected %s, got %s", testContent, data)
	}

}

func TestMemoryReadlink(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testLink := "link"
	testPathNotExist := "notexist"

	// create a symlink
	if err := tm.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// read the symlink
	link, err := tm.Readlink(testLink)
	if err != nil {
		t.Fatalf("Readlink() failed: %s", err)
	}

	if link != testPath {
		t.Fatalf("Readlink() failed: expected %s, got %s", testPath, link)
	}

	// read a symlink that does not exist
	if _, err := tm.Readlink(testPathNotExist); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// readlink a file
	if _, err := tm.Readlink(testPath); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}
}

func TestMemoryWithFsTest(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "foo"
	testContent := "hello world"
	testLink := "bar"
	testDir := "baz"
	testDirFile := "baz/qux"

	expectedFiles := []string{testPath, testLink, testDir, testDirFile}

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0640, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create a symlink
	if err := tm.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// create directory
	if err := tm.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create file with directory
	if _, err := tm.CreateFile(testDirFile, bytes.NewReader([]byte(testContent)), 0640, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// perform test
	if err := fstest.TestFS(tm, expectedFiles...); err != nil {
		t.Fatalf("TestFS() failed: %s", err)
	}

}

func TestMemoryOpen2(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPathNotExist := "notexist"
	testDir := "dir"

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// open the file
	f, err := tm.Open(testPath)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}
	defer f.Close()

	// read the file
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() failed: %s", err)
	}

	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("ReadAll() failed: expected %s, got %s", testContent, data)
	}

	// open a file that does not exist
	if _, err := tm.Open(testPathNotExist); err == nil {
		t.Fatalf("Open() failed: expected error, got nil")
	}

	// create a directory
	if err := tm.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// open the directory
	if _, err := tm.Open(testDir); err != nil {
		t.Fatalf("Open() failed: %s", err)
	}
}

func TestMemoryLstat(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testPathNotExist := "notexist"

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// lstat the file
	stat, err := tm.Lstat(testPath)
	if err != nil {
		t.Fatalf("Lstat() failed: %s", err)
	}

	// check the file permissions
	if int(stat.Mode().Perm()) != 0644 {
		t.Fatalf("Lstat() failed: expected 0644, got %d", stat.Mode().Perm())
	}

	// lstat a symlink that does not exist
	if _, err := tm.Lstat(testPathNotExist); err == nil {
		t.Fatalf("Lstat() failed: expected error, got nil")
	}
}

func TestMemoryStat(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testInvalidPath := "../test/invalid"
	testLink := "link"
	testPathNotExist := "notexist"

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// stat the file
	stat, err := tm.Stat(testPath)
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check the file permissions
	if int(stat.Mode().Perm()) != 0644 {
		t.Fatalf("Stat() failed: expected 0644, got %d", stat.Mode().Perm())
	}

	// stat a file with invalid path
	if _, err := tm.Stat(testInvalidPath); err == nil {
		t.Fatalf("Stat() failed: expected error, got nil")
	}

	// create a symlink
	if err := tm.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// stat the symlink
	stat, err = tm.Stat(testLink)
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check the file permissions
	if int(stat.Mode().Perm()) != 0644 {
		t.Fatalf("Stat() failed: expected 0644, got %d", stat.Mode().Perm())
	}

	// stat a symlink that does not exist
	if _, err := tm.Stat(testPathNotExist); err == nil {
		t.Fatalf("Stat() failed: expected error, got nil")
	}
}

func TestMemoryRemove(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testPathNotExist := "notexist"
	testDir := "dir"
	testDirFile := "dir/file"

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// remove the file
	if err := tm.Remove(testPath); err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

	// remove a file that does not exist
	if err := tm.Remove(testPathNotExist); err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

	// create a directory
	if err := tm.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create a file in the directory
	if _, err := tm.CreateFile(testDirFile, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// remove the directory
	if err := tm.Remove(testDir); err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

	// remove a file thats open and expect to fail
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() for testPath failed: %s", err)
	}
	f, err := tm.Open(testPath)
	if err != nil {
		t.Fatalf("Open() testPath failed: %s", err)
	}
	err = tm.Remove(testPath)
	if err == nil {
		t.Fatalf("Remove() misbehaved failed: expected error, got nil")
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() failed: %s", err)
	}
	err = tm.Remove(testPath)
	if err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

}

func TestMemoryReadDir(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testDir := "dir"
	testDir2 := "dir2"
	testPathNotExist := "notexist"

	// create a directory
	if err := tm.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create an additional directory
	if err := tm.CreateDir(testDir2, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// read the root
	entries, err := tm.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir() failed: %s", err)
	}

	if len(entries) != 2 {
		t.Fatalf("ReadDir() failed: expected 2, got %d", len(entries))
	}

	// read the directory
	entries, err = tm.ReadDir(testDir)
	if err != nil {
		t.Fatalf("ReadDir() failed: %s", err)
	}

	if len(entries) != 0 {
		t.Fatalf("ReadDir() failed: expected 0, got %d", len(entries))
	}

	// read a directory that does not exist
	if _, err := tm.ReadDir(testPathNotExist); err == nil {
		t.Fatalf("ReadDir() failed: expected error, got nil")
	}
}

func TestMemoryReadFile(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPathNotExist := "notexist"
	testDir := "dir"

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// read the file
	data, err := tm.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile() failed: %s", err)
	}

	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("ReadFile() failed: expected %s, got %s", testContent, data)
	}

	// read a file that does not exist
	if _, err := tm.ReadFile(testPathNotExist); err == nil {
		t.Fatalf("ReadFile() failed: expected error, got nil")
	}

	// create a directory
	if err := tm.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// read a directory
	if _, err := tm.ReadFile(testDir); err == nil {
		t.Fatalf("ReadFile() failed: expected error, got nil")
	}
}

func TestMemorySub(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testDir := "dir"
	testSubDir := "subdir"
	testPathNotExist := "notexist"

	// create a directory
	if err := tm.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create an additional directory
	if err := tm.CreateDir(p.Join(testDir, testSubDir), 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// sub the root
	sub, err := tm.Sub(testDir)
	if err != nil {
		t.Fatalf("Sub() failed: %s", err)
	}

	// read the sub
	subFs, ok := sub.(*extract.TargetMemory)
	if !ok {
		t.Fatalf("Sub() failed: expected Memory, got %T", sub)
	}
	entries, err := subFs.ReadDir(".")
	if err != nil {
		t.Fatalf("[%v].ReadDir(.) failed: %s", subFs, err)
	}

	if len(entries) != 1 {
		t.Fatalf("[%v].ReadDir(.) failed: expected 1, got %d", subFs, len(entries))
	}

	// sub a directory that does not exist
	if _, err := tm.Sub(testPathNotExist); err == nil {
		t.Fatalf("Sub() failed: expected error, got nil")
	}
}

func TestMemoryGlob(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testPath2 := "test2"
	testPattern := "test*"

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create an additional file
	if _, err := tm.CreateFile(testPath2, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// glob the files
	matches, err := tm.Glob(testPattern)
	if err != nil {
		t.Fatalf("Glob() failed: %s", err)
	}

	if len(matches) != 2 {
		t.Fatalf("Glob() failed: expected 2, got %d", len(matches))
	}
}

func TestInvalidPath(t *testing.T) {
	// instantiate a new memory target
	tm := extract.NewTargetMemory()

	// test data
	invalidPath := "../invalid/path"

	// create a file
	if _, err := tm.CreateFile(invalidPath, bytes.NewReader([]byte("test")), 0644, false, -1); err == nil {
		t.Fatalf("CreateFile(%s) failed: expected error, got nil", invalidPath)
	}

	// create a directory
	if err := tm.CreateDir(invalidPath, 0755); err == nil {
		t.Fatalf("CreateDir() failed: expected error, got nil")
	}

	// create a symlink
	if err := tm.CreateSymlink("target", invalidPath, false); err == nil {
		t.Fatalf("CreateSymlink() failed: expected error, got nil")
	}

	// open the file
	if _, err := tm.Open(invalidPath); err == nil {
		t.Fatalf("Open() failed: expected error, got nil")
	}

	// lstat the file
	if _, err := tm.Lstat(invalidPath); err == nil {
		t.Fatalf("Lstat() failed: expected error, got nil")
	}

	// stat the file
	if _, err := tm.Stat(invalidPath); err == nil {
		t.Fatalf("Stat() failed: expected error, got nil")
	}

	// readlink the file
	if _, err := tm.Readlink(invalidPath); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}

	// remove the file
	if err := tm.Remove(invalidPath); err == nil {
		t.Fatalf("Remove() failed: expected error, got nil")
	}

	// read the file
	if _, err := tm.ReadFile(invalidPath); err == nil {
		t.Fatalf("ReadFile() failed: expected error, got nil")
	}

	// readdir
	if _, err := tm.ReadDir(invalidPath); err == nil {
		t.Fatalf("ReadDir() failed: expected error, got nil")
	}

	// sub the file
	if _, err := tm.Sub(invalidPath); err == nil {
		t.Fatalf("Sub() failed: expected error, got nil")
	}

	// glob the file
	if _, err := tm.Glob(invalidPath); err == nil {
		t.Fatalf("Glob() failed: expected error, got nil")
	}
}

func TestMemoryEntry(t *testing.T) {
	// instantiate a new memory
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPerm := 0644

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// open the file
	f, err := tm.Open(testPath)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}

	// is dir
	if f.(fs.DirEntry).IsDir() {
		t.Fatalf("IsDir() returned unexpected value: expected false, got true")
	}

	// type
	if f.(fs.DirEntry).Type() != 0 {
		t.Fatalf("Type() returned unexpected value: expected 0, got %d", f.(fs.DirEntry).Type())
	}

	// stat the file
	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check isDir
	if stat.IsDir() {
		t.Fatalf("IsDir() returned unexpected value: expected false, got true")
	}

	// check name
	if stat.Name() != testPath {
		t.Fatalf("Name() returned unexpected value: expected %s, got %s", testPath, stat.Name())
	}

	// check mode
	if int(stat.Mode().Perm()&fs.ModePerm) != testPerm {
		t.Fatalf("Mode() returned unexpected value: expected %d, got %d", testPerm, stat.Mode().Perm())
	}

	// check type
	if stat.Mode().Type() != 0 {
		t.Fatalf("Type() failed: expected 0, got %d", stat.Mode().Type())
	}

	// check info
	de, err := f.(fs.DirEntry).Info()
	if err != nil {
		t.Fatalf("Info() failed: %s", err)
	}

	if de != stat {
		t.Fatalf("Info() returned unexpected value: expected %v, got %v", stat, de)
	}

	// check size
	if stat.Size() != int64(len(testContent)) {
		t.Fatalf("Size() returned unexpected value: expected %d, got %d", len(testContent), stat.Size())
	}

	// modtime
	if stat.ModTime().IsZero() {
		t.Fatalf("unexpected ModTime() value: expected non-zero, got zero")
	}

	// check sys
	if stat.Sys() != nil {
		t.Fatalf("unexpected return value from Sys(): expected nil, got %v", stat.Sys())
	}

	// read the file
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() failed: %s", err)
	}

	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("unexpected file contents: expected %s, got %s", testContent, data)
	}

	// close the file
	if err := f.Close(); err != nil {
		t.Fatalf("Close() failed: %s", err)
	}
}

func TestCreateFile(t *testing.T) {
	// instantiate a new memory
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPerm := 0644

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), fs.FileMode(testPerm), false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create the same file, but fail bc it already exists#
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), fs.FileMode(testPerm), false, -1); err == nil {
		t.Fatalf("CreateFile() failed: expected error, got nil")
	}

	// create the same file, but overwrite
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), fs.FileMode(testPerm), true, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// open the file
	f, err := tm.Open(testPath)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}

	// stat the file
	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check name
	if stat.Name() != testPath {
		t.Fatalf("Name() returned unexpected value: expected %s, got %s", testPath, stat.Name())
	}

	// check mode
	if int(stat.Mode().Perm()&fs.ModePerm) != testPerm {
		t.Fatalf("Mode() returned unexpected value: expected %d, got %d", testPerm, stat.Mode().Perm())
	}

	// read the file
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() failed: %s", err)
	}

	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("unexpected file contents: expected %s, got %s", testContent, data)
	}

	// close the file
	if err := f.Close(); err != nil {
		t.Fatalf("Close() failed: %s", err)
	}

	// create a dir
	testDir := "dir"
	if err := tm.CreateDir(testDir, fs.FileMode(testPerm)); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create a file with the same name as the dir
	if _, err := tm.CreateFile(testDir, bytes.NewReader([]byte(testContent)), fs.FileMode(testPerm), false, -1); err == nil {
		t.Fatalf("CreateFile() failed: expected error, got nil")
	}
}

// TestCreateSymlink tests the CreateSymlink method
func TestCreateSymlink(t *testing.T) {
	// instantiate a new memory
	tm := extract.NewTargetMemory()

	// test data
	testPath := "test"
	testLink := "link"
	testContent := "test"
	testPerm := 0644

	// create a file
	if _, err := tm.CreateFile(testPath, bytes.NewReader([]byte(testContent)), fs.FileMode(testPerm), false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create a symlink
	if err := tm.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// open the symlink
	f, err := tm.Open(testLink)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}

	// stat the symlink (which is the link target)
	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check name
	if stat.Name() != testPath {
		t.Fatalf("Name() returned unexpected value: expected %s, got %s", testLink, stat.Name())
	}

	// check mode
	if int(stat.Mode().Perm()&fs.ModePerm) != testPerm {
		t.Fatalf("Mode() returned unexpected value: expected %d, got %d", testPerm, stat.Mode().Perm())
	}

	// read the symlink
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() failed: %s", err)
	}
	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("unexpected file contents: expected %s, got %s", testContent, data)
	}

	// close the symlink
	if err := f.Close(); err != nil {
		t.Fatalf("Close() failed: %s", err)
	}

	// overwrite the symlink, but fail
	if err := tm.CreateSymlink(testPath, testLink, false); err == nil {
		t.Fatalf("CreateSymlink() failed: expected error, got nil")
	}

	// overwrite the symlink
	if err := tm.CreateSymlink(testPath, testLink, true); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}
}

func TestUnpackToMemoryWithPreserveFileAttributes(t *testing.T) {
	uid, gid := 503, 20
	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	testCases := []struct {
		name                  string
		contents              []archiveContent
		packer                func(*testing.T, []archiveContent) []byte
		doesNotSupportModTime bool
		expectError           bool
	}{
		{
			name: "unpack tar with preserve file attributes",
			contents: []archiveContent{
				{Name: "test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				// {Name: "sub", Mode: fs.ModeDir | 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "sub/test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "link", Mode: fs.ModeSymlink | 0777, Linktarget: "sub/test", AccessTime: baseTime, ModTime: baseTime},
			},
			packer: packTar,
		},
		{
			name: "unpack zip with preserve file attributes",
			contents: []archiveContent{
				{Name: "test", Content: []byte("hello world"), Mode: 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				// {Name: "sub", Mode: fs.ModeDir | 0777, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "sub/test", Content: []byte("hello world"), Mode: 0644, AccessTime: baseTime, ModTime: baseTime, Uid: uid, Gid: gid},
				{Name: "link", Mode: fs.ModeSymlink | 0777, Linktarget: "sub/test", AccessTime: baseTime, ModTime: baseTime},
			},
			packer: packZip,
		},
		{
			name:                  "unpack rar with preserve file attributes",
			contents:              contentsRar2,
			doesNotSupportModTime: true,
			packer:                packRar2,
		},
		{
			name:     "unpack z7 with preserve file attributes",
			contents: contents7z2,
			packer:   pack7z2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ctx = context.Background()
				m   = extract.NewTargetMemory()
				src = asIoReader(t, tc.packer(t, tc.contents))
				cfg = extract.NewConfig(extract.WithPreserveFileAttributes(true))
			)
			if err := extract.UnpackTo(ctx, m, "", src, cfg); err != nil {
				t.Fatalf("error unpacking archive: %v", err)
			}

			for _, c := range tc.contents {
				parts := strings.Split(c.Name, "/") // create system specific path
				path := filepath.Join(parts...)
				stat, err := m.Lstat(path)
				if err != nil {
					t.Fatalf("error getting file stats: %v", err)
				}
				if !(c.Mode&fs.ModeSymlink != 0) { // skip symlink checks
					if stat.Mode().Perm() != c.Mode.Perm() {
						t.Fatalf("expected file mode %v, got %v, file %s", c.Mode.Perm(), stat.Mode().Perm(), path)
					}
				}
				if tc.doesNotSupportModTime {
					continue
				}
				// calculate the time difference
				modTimeDiff := abs(stat.ModTime().UnixNano() - c.ModTime.UnixNano())
				if modTimeDiff >= int64(time.Microsecond) {
					t.Fatalf("expected file modtime %v, got %v, file %s, diff %v", int64(c.ModTime.UnixMicro()), int64(stat.ModTime().UnixMicro()), path, modTimeDiff)
				}
			}
		})
	}
}
