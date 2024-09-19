package target

import (
	"bytes"
	"io"
	"io/fs"
	p "path"
	"testing"
	"testing/fstest"
)

func TestMemoryOpen(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPerm := 0644
	testNotExist := "notexist"
	testDir := "dir"
	testLink := "link"

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// open the file
	f, err := mem.Open(testPath)
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
	if _, err := mem.Open(testNotExist); err == nil {
		t.Fatalf("Open() failed: expected error, got nil")
	}

	// create a directory
	if err := mem.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("failed to perform CreateDir(): %s", err)
	}

	// open the directory
	if _, err := mem.Open(testDir); err != nil {
		t.Fatalf("failed to Open() directory: %s", err)
	}

	// create a symlink
	if err := mem.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// open the symlink
	f, err = mem.Open(testLink)
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

func TestMemoryWithFsTest(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "foo"
	testContent := "hello world"
	testLink := "bar"
	testDir := "baz"
	testDirFile := "baz/qux"

	expectedFiles := []string{testPath, testLink, testDir, testDirFile}

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0640, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create a symlink
	if err := mem.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// create directory
	if err := mem.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create file with directory
	if _, err := mem.CreateFile(testDirFile, bytes.NewReader([]byte(testContent)), 0640, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// perform test
	if err := fstest.TestFS(mem); err == nil {
		t.Fatalf("TestFS() failed: %s", err)
	}

	// perform test
	if err := fstest.TestFS(mem, expectedFiles...); err != nil {
		t.Fatalf("TestFS() failed: %s", err)
	}

}

// TestMemoryOpen
func TestMemoryOpen2(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPathNotExist := "notexist"
	testDir := "dir"

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// open the file
	f, err := mem.Open(testPath)
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
	if _, err := mem.Open(testPathNotExist); err == nil {
		t.Fatalf("Open() failed: expected error, got nil")
	}

	// create a directory
	if err := mem.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// open the directory
	if _, err := mem.Open(testDir); err != nil {
		t.Fatalf("Open() failed: %s", err)
	}
}

// TestMemoryLstat tests the Lstat function from Memory
func TestMemoryLstat(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testPathNotExist := "notexist"

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// lstat the file
	stat, err := mem.Lstat(testPath)
	if err != nil {
		t.Fatalf("Lstat() failed: %s", err)
	}

	// check the file permissions
	if int(stat.Mode().Perm()) != 0644 {
		t.Fatalf("Lstat() failed: expected 0644, got %d", stat.Mode().Perm())
	}

	// lstat a symlink that does not exist
	if _, err := mem.Lstat(testPathNotExist); err == nil {
		t.Fatalf("Lstat() failed: expected error, got nil")
	}
}

// TestMemoryStat tests the Stat function from Memory
func TestMemoryStat(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testInvalidPath := "../test/invalid"
	testLink := "link"
	testPathNotExist := "notexist"

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// stat the file
	stat, err := mem.Stat(testPath)
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check the file permissions
	if int(stat.Mode().Perm()) != 0644 {
		t.Fatalf("Stat() failed: expected 0644, got %d", stat.Mode().Perm())
	}

	// stat a file with invalid path
	if _, err := mem.Stat(testInvalidPath); err == nil {
		t.Fatalf("Stat() failed: expected error, got nil")
	}

	// create a symlink
	if err := mem.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// stat the symlink
	stat, err = mem.Stat(testLink)
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check the file permissions
	if int(stat.Mode().Perm()) != 0644 {
		t.Fatalf("Stat() failed: expected 0644, got %d", stat.Mode().Perm())
	}

	// stat a symlink that does not exist
	if _, err := mem.Stat(testPathNotExist); err == nil {
		t.Fatalf("Stat() failed: expected error, got nil")
	}
}

// TestMemoryReadlink tests the Readlink function from Memory
func TestMemoryReadlink(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testLink := "link"
	testPathNotExist := "notexist"

	// create a symlink
	if err := mem.CreateSymlink(testPath, testLink, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// read the symlink
	link, err := mem.Readlink(testLink)
	if err != nil {
		t.Fatalf("Readlink() failed: %s", err)
	}

	if link != testPath {
		t.Fatalf("Readlink() failed: expected %s, got %s", testPath, link)
	}

	// read a symlink that does not exist
	if _, err := mem.Readlink(testPathNotExist); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// readlink a file
	if _, err := mem.Readlink(testPath); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}
}

// TestMemoryRemove tests the Remove function from Memory
func TestMemoryRemove(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testPathNotExist := "notexist"
	testDir := "dir"
	testDirFile := "dir/file"

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// remove the file
	if err := mem.Remove(testPath); err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

	// remove a file that does not exist
	if err := mem.Remove(testPathNotExist); err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

	// create a directory
	if err := mem.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create a file in the directory
	if _, err := mem.CreateFile(testDirFile, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// remove the directory
	if err := mem.Remove(testDir); err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

}

// TestMemoryReadDir tests the ReadDir function from Memory
func TestMemoryReadDir(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testDir := "dir"
	testDir2 := "dir2"
	testPathNotExist := "notexist"

	// create a directory
	if err := mem.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create an additional directory
	if err := mem.CreateDir(testDir2, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// read the root
	entries, err := mem.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir() failed: %s", err)
	}

	if len(entries) != 2 {
		t.Fatalf("ReadDir() failed: expected 2, got %d", len(entries))
	}

	// read the directory
	entries, err = mem.ReadDir(testDir)
	if err != nil {
		t.Fatalf("ReadDir() failed: %s", err)
	}

	if len(entries) != 0 {
		t.Fatalf("ReadDir() failed: expected 0, got %d", len(entries))
	}

	// read a directory that does not exist
	if _, err := mem.ReadDir(testPathNotExist); err == nil {
		t.Fatalf("ReadDir() failed: expected error, got nil")
	}
}

// TestMemoryReadFile tests the ReadFile function from Memory
func TestMemoryReadFile(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPathNotExist := "notexist"
	testDir := "dir"

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// read the file
	data, err := mem.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile() failed: %s", err)
	}

	if !bytes.Equal(data, []byte(testContent)) {
		t.Fatalf("ReadFile() failed: expected %s, got %s", testContent, data)
	}

	// read a file that does not exist
	if _, err := mem.ReadFile(testPathNotExist); err == nil {
		t.Fatalf("ReadFile() failed: expected error, got nil")
	}

	// create a directory
	if err := mem.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// read a directory
	if _, err := mem.ReadFile(testDir); err == nil {
		t.Fatalf("ReadFile() failed: expected error, got nil")
	}
}

// TestMemorySub tests the Sub function from Memory
func TestMemorySub(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testDir := "dir"
	testSubDir := "subdir"
	testPathNotExist := "notexist"

	// create a directory
	if err := mem.CreateDir(testDir, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// create an additional directory
	if err := mem.CreateDir(p.Join(testDir, testSubDir), 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// sub the root
	sub, err := mem.Sub(testDir)
	if err != nil {
		t.Fatalf("Sub() failed: %s", err)
	}

	// read the sub
	subFs, ok := sub.(*Memory)
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
	if _, err := mem.Sub(testPathNotExist); err == nil {
		t.Fatalf("Sub() failed: expected error, got nil")
	}
}

// TestMemoryGlob tests the Glob function from Memory
func TestMemoryGlob(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	testPath := "test"
	testPath2 := "test2"
	testPattern := "test*"

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create an additional file
	if _, err := mem.CreateFile(testPath2, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// glob the files
	matches, err := mem.Glob(testPattern)
	if err != nil {
		t.Fatalf("Glob() failed: %s", err)
	}

	if len(matches) != 2 {
		t.Fatalf("Glob() failed: expected 2, got %d", len(matches))
	}
}

// TestInvalidPath tests the ValidPath check on every function from Memory
func TestInvalidPath(t *testing.T) {

	// instantiate a new memory target
	mem := NewMemory()

	// test data
	invalidPath := "../invalid/path"

	// create a file
	if _, err := mem.CreateFile(invalidPath, bytes.NewReader([]byte("test")), 0644, false, -1); err == nil {
		t.Fatalf("CreateFile(%s) failed: expected error, got nil", invalidPath)
	}

	// create a directory
	if err := mem.CreateDir(invalidPath, 0755); err == nil {
		t.Fatalf("CreateDir() failed: expected error, got nil")
	}

	// create a symlink
	if err := mem.CreateSymlink("target", invalidPath, false); err == nil {
		t.Fatalf("CreateSymlink() failed: expected error, got nil")
	}

	// open the file
	if _, err := mem.Open(invalidPath); err == nil {
		t.Fatalf("Open() failed: expected error, got nil")
	}

	// lstat the file
	if _, err := mem.Lstat(invalidPath); err == nil {
		t.Fatalf("Lstat() failed: expected error, got nil")
	}

	// stat the file
	if _, err := mem.Stat(invalidPath); err == nil {
		t.Fatalf("Stat() failed: expected error, got nil")
	}

	// readlink the file
	if _, err := mem.Readlink(invalidPath); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}

	// remove the file
	if err := mem.Remove(invalidPath); err == nil {
		t.Fatalf("Remove() failed: expected error, got nil")
	}

	// read the file
	if _, err := mem.ReadFile(invalidPath); err == nil {
		t.Fatalf("ReadFile() failed: expected error, got nil")
	}

	// readdir
	if _, err := mem.ReadDir(invalidPath); err == nil {
		t.Fatalf("ReadDir() failed: expected error, got nil")
	}

	// sub the file
	if _, err := mem.Sub(invalidPath); err == nil {
		t.Fatalf("Sub() failed: expected error, got nil")
	}

	// glob the file
	if _, err := mem.Glob(invalidPath); err == nil {
		t.Fatalf("Glob() failed: expected error, got nil")
	}
}

// TestMemoryEntry tests the MemoryEntry functions
func TestMemoryEntry(t *testing.T) {

	// instantiate a new memory
	mem := NewMemory()

	// test data
	testPath := "test"
	testContent := "test"
	testPerm := 0644

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte(testContent)), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// open the file
	f, err := mem.Open(testPath)
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
