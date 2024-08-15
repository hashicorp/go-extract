package target

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

var testTargets = []Target{
	NewOS(),
	NewMemory(),
}

// TestCreateDir tests the CreateDir function from Os
func TestCreateDir(t *testing.T) {

	for _, tt := range testTargets {

		// prepare tmp
		tmp := t.TempDir()

		// create a directory
		testDir := "test"
		testPath := filepath.Join(tmp, testDir)
		if err := tt.CreateDir(testPath, 0755); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}

		// check if directory exists
		if _, err := tt.Lstat(testPath); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}

		// create a directory that already exists
		if err := tt.CreateDir(testPath, 0755); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}
	}
}

// TestMemoryAccessContent tests that the decompressed content from an archive can be accessed
func TestMemoryAccessContent(t *testing.T) {

	content := []byte("test data")
	testFileName := "test.txt"
	testLinkName := "test_link"
	testDirName := "test_dir"

	// create a memory target
	mem := NewMemory()

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
	e, found := mem[testFileName]
	if !found {
		t.Fatalf("entry not found")
	}

	// check size of entry
	if e.FileInfo.Size() != int64(len(content)) {
		t.Fatalf("size mismatch: %d", e.FileInfo.Size())
	}

	// get mod time of entry ; should be zero
	modTime := e.FileInfo.ModTime()
	if modTime.IsZero() {
		t.Fatalf("mod time is zero")
	}

	// get mode of entry
	mode := e.FileInfo.Mode()
	if mode&fs.ModeType != 0 { // check if it is a file
		t.Fatalf("mode mismatch: %s", mode)
	}

	// check Sys() of entry
	if e.FileInfo.Sys() != nil {
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

}

// TestCreateFile tests the CreateFile function from Os
func TestCreateFile(t *testing.T) {

	for _, tt := range testTargets {

		// prepare tmp and ensure that tmp dir exist even on mem target
		tmp := t.TempDir()
		if err := tt.CreateDir(tmp, 0755); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}

		// prepare file details
		testFile := "test"
		testPath := filepath.Join(tmp, testFile)
		testData := []byte("test data")
		testReader := bytes.NewReader(testData)

		// create a file
		if _, err := tt.CreateFile(testPath, testReader, 0644, false, -1); err != nil {
			t.Fatalf("CreateFile() failed: %s", err)
		}

		// check if file exists
		if _, err := tt.Lstat(testPath); err != nil {
			t.Fatalf("CreateFile() failed: %s", err)
		}

		// create a file with overwrite
		if _, err := tt.CreateFile(testPath, testReader, 0644, true, -1); err != nil {
			t.Fatalf("CreateFile() with overwrite failed: %s", err)
		}
		if _, err := testReader.Seek(0, 0); err != nil {
			t.Fatalf("failed to set testReader: %s", err)
		}
		// create a file with overwrite expect fail
		if _, err := tt.CreateFile(testPath, testReader, 0644, false, -1); err == nil {
			t.Fatalf("CreateFile() with disabled overwrite try to overwrite failed: %s", err)
		}
		if _, err := testReader.Seek(0, 0); err != nil {
			t.Fatalf("failed to set testReader: %s", err)
		}

		// create a file with maxSize
		if n, err := tt.CreateFile(testPath, testReader, 0644, true, 5); err == nil {
			t.Fatalf("CreateFile() with maxSize failed: err: %s, n: %v", err, n)
		}
	}

}

// TestCreateSymlink tests the CreateSymlink function from Os
func TestCreateSymlink(t *testing.T) {

	for _, tt := range testTargets {

		// prepare tmp and ensure that tmp dir exist even on mem target
		tmp := t.TempDir()
		if err := tt.CreateDir(tmp, 0755); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}

		// prepare link details
		testFile := "test"
		testSymlink := "symlink"
		testPath := filepath.Join(tmp, testFile)
		testSymlinkPath := filepath.Join(tmp, testSymlink)
		testData := []byte("test data")
		testReader := bytes.NewReader(testData)

		// create a file
		if _, err := tt.CreateFile(testPath, testReader, 0644, false, -1); err != nil {
			t.Fatalf("CreateFile() failed: %s", err)
		}

		// create a symlink

		if err := tt.CreateSymlink(testPath, testSymlinkPath, false); err != nil {
			t.Fatalf("CreateSymlink() failed: %s", err)
		}

		// check if symlink exists
		lstat, err := tt.Lstat(testSymlinkPath)
		if err != nil {
			t.Fatalf("CreateSymlink() failed: %s", err)
		}
		if lstat.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("CreateSymlink() failed: %s", "not a symlink")
		}

		// create a symlink with overwrite
		if err := tt.CreateSymlink(testSymlinkPath, testPath, true); err != nil {
			t.Fatalf("CreateSymlink() with overwrite failed: %s", err)
		}

		// create a symlink with overwrite expect fail
		if err := tt.CreateSymlink(testSymlinkPath, testPath, false); err == nil {
			t.Fatalf("CreateSymlink() with disabled overwrite try to overwrite failed: %s", err)
		}
	}

}
