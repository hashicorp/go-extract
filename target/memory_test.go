package target

import (
	"bytes"
	"io/fs"
	"testing"
)

// TestMemoryAccessContent tests that the decompressed content from an archive can be accessed
func TestMemoryAccessContent(t *testing.T) {

	content := []byte("test data")
	testFileName := "test.txt"
	testLinkName := "test_link"
	testDirName := "test_dir"

	// create a memory target
	mem := NewMemory().(Memory)

	// create a file
	if _, err := mem.CreateFile(testFileName, bytes.NewReader(content), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create a symlink
	if err := mem.CreateSymlink(testFileName, testLinkName, false); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// create a directory
	if err := mem.CreateDir(testDirName, 0755); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// check if file exists
	if _, err := mem.Stat(testFileName); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// check if symlink exists
	if _, err := mem.Lstat(testLinkName); err != nil {
		t.Fatalf("CreateSymlink() failed: %s", err)
	}

	// check if directory exists
	if _, err := mem.Lstat(testDirName); err != nil {
		t.Fatalf("CreateDir() failed: %s", err)
	}

	// check if file content is correct
	file, err := mem.Open(testFileName)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}
	defer file.Close()

	// check if file content is correct
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		t.Fatalf("ReadFrom() failed: %s", err)
	}
	if !bytes.Equal(buf.Bytes(), content) {
		t.Fatalf("content mismatch: %s", buf.Bytes())
	}

	// check if stat follows symlink
	link, err := mem.Stat(testLinkName)
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}
	if link.Name() != testFileName {
		t.Fatalf("link mismatch: %s", link.Name())
	}

	// get entry from memory
	tFile, err := mem.Open(testFileName)
	if err != nil {
		t.Fatalf("entry not found")
	}
	defer tFile.Close()

	// get stat
	tStat, err := tFile.Stat()
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}

	// check size of entry
	if s := tStat.Size(); s != int64(len(content)) {
		t.Fatalf("size mismatch: %d != %d", s, len(content))
	}

	// get mod time of entry ; should be zero
	modTime := tStat.ModTime()
	if modTime.IsZero() {
		t.Fatalf("mod time is zero")
	}

	// get mode of entry
	mode := tStat.Mode()
	if mode&fs.ModeType != 0 { // check if it is a file
		t.Fatalf("mode mismatch: %s", mode)
	}

	// check Sys() of entry
	if tStat.Sys() != nil {
		t.Fatalf("Sys() should be nil")
	}

	// check if file is a directory
	dir, err := mem.Stat(testDirName)
	if err != nil {
		t.Fatalf("Stat() failed: %s", err)
	}
	if dir.Name() != testDirName {
		t.Fatalf("dir mismatch: %s", dir.Name())
	}
	if dir.IsDir() == false {
		t.Fatalf("dir is not a directory")
	}

	// stat for non-existing file
	if _, err := mem.Stat("non-existing"); err == nil {
		t.Fatalf("Stat() should fail")
	}

	// lstat for non-existing file
	if _, err := mem.Lstat("non-existing"); err == nil {
		t.Fatalf("Lstat() should fail")
	}

	// open for non-existing file
	if _, err := mem.Open("non-existing"); err == nil {
		t.Fatalf("Open() should fail")
	}

	// read dir
	entries, err := fs.ReadDir(mem, ".")
	if err != nil {
		t.Fatalf("ReadDir() failed: %s", tFile)
	}
	if len(entries) != 3 {
		t.Fatalf("ReadDir() failed: %d", len(entries))
	}
	t.Logf("entries: %v", entries)

	// get file and modify data maliciously
	fileEntry, err := mem.Open(testFileName)
	if err != nil {
		t.Fatalf("Open() failed: %s", err)
	}
	defer fileEntry.Close()
	fileEntry.(*MemoryEntry).Data = []byte("modified data")

	// test readfile
	data, err := fs.ReadFile(mem, testFileName)
	if err != nil {
		t.Fatalf("ReadFile() failed: %s", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("content mismatch: %s != %s", data, content)
	}
	// read agin
	dataSame, err := fs.ReadFile(mem, testFileName)
	if err != nil {
		t.Fatalf("ReadFile() failed: %s", err)
	}
	if !bytes.Equal(dataSame, data) {
		t.Fatalf("content mismatch after second read: %s != %s", dataSame, data)
	}

	// remove file
	if err := mem.Remove(testFileName); err != nil {
		t.Fatalf("Remove() failed: %s", err)
	}

}
