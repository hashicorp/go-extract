package target

import (
	"bytes"
	"path/filepath"
	"testing"
)

var testTargets = []Target{
	NewOS(),
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

		// create a directory with 0000 mode and expect fail
		if err := tt.CreateDir(filepath.Join(testPath, "foo", "bar", "baz"), 0000); err == nil {
			t.Fatalf("CreateDir() did not fail, but error was expected!")
		}
	}

}

// TestCreateFile tests the CreateFile function from Os
func TestCreateFile(t *testing.T) {

	for _, tt := range testTargets {

		// prepare tmp and ensure that tmp dir exist even on mem target
		tmp := t.TempDir()
		tt.CreateDir(tmp, 0755)

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
		tt.CreateDir(tmp, 0755)

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
		if _, err := tt.Lstat(testSymlinkPath); err != nil {
			t.Fatalf("CreateSymlink() failed: %s", err)
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
