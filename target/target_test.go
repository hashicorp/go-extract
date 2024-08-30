package target

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type TargetFunc func() Target

// list of function to create new targets
var testTargets = []TargetFunc{
	NewOS,
	NewMemory,
}

// var testTargets = [](func() *Target){
// 	&NewOS(),
// 	&NewMemory(),
// }

// TestCreateDir tests the CreateDir function from Os
func TestCreateDir(t *testing.T) {

	for _, tt := range testTargets {

		target := tt()

		testPath := "test"

		// create tmp dir if target is os
		if _, ok := target.(*OS); ok {
			tmp := t.TempDir()
			testPath = filepath.Join(tmp, testPath)
		}

		// create a directory
		if err := target.CreateDir(testPath, 0755); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}

		// check if directory exists
		if _, err := target.Lstat(testPath); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}

		// create a directory that already exists
		if err := target.CreateDir(testPath, 0755); err != nil {
			t.Fatalf("CreateDir() failed: %s", err)
		}
	}
}

// TestCreateFile tests the CreateFile function from Os
func TestCreateFile(t *testing.T) {

	for _, tt := range testTargets {

		target := tt()

		// test file details
		testFile := "test"
		testData := []byte("test data")
		testReader := bytes.NewReader(testData)

		// create tmp dir if target is os
		if _, ok := target.(*OS); ok {
			tmp := t.TempDir()
			testFile = filepath.Join(tmp, testFile)
		}

		// create a file
		if _, err := target.CreateFile(testFile, testReader, 0644, false, -1); err != nil {
			t.Fatalf("1 CreateFile() failed: %s", err)
		}

		// check if file exists
		if _, err := target.Lstat(testFile); err != nil {
			t.Fatalf("2 CreateFile() failed: %s", err)
		}

		// create a file with overwrite
		if _, err := target.CreateFile(testFile, testReader, 0644, true, -1); err != nil {
			t.Fatalf("3 CreateFile() with overwrite failed: %s", err)
		}
		if _, err := testReader.Seek(0, 0); err != nil {
			t.Fatalf("failed to set testReader: %s", err)
		}
		// create a file with overwrite expect fail
		if _, err := target.CreateFile(testFile, testReader, 0644, false, -1); err == nil {
			t.Fatalf("CreateFile() with disabled overwrite try to overwrite failed: %s", err)
		}
		if _, err := testReader.Seek(0, 0); err != nil {
			t.Fatalf("failed to set testReader: %s", err)
		}

		// create a file with maxSize
		if n, err := target.CreateFile(testFile, testReader, 0644, true, 5); err == nil {
			t.Fatalf("CreateFile() with maxSize failed: err: %s, n: %v", err, n)
		}
	}

}

// TestCreateSymlink tests the CreateSymlink function from Os
func TestCreateSymlink(t *testing.T) {

	for _, tt := range testTargets {

		target := tt()

		// prepare test data and link details
		testFile := "test"
		testSymlink := "symlink"
		testData := []byte("test data")
		testReader := bytes.NewReader(testData)

		// create tmp dir if target is os
		if _, ok := target.(*OS); ok {
			tmp := t.TempDir()
			testFile = filepath.Join(tmp, testFile)
			testSymlink = filepath.Join(tmp, testSymlink)
		}

		// create a file
		if _, err := target.CreateFile(testFile, testReader, 0644, false, -1); err != nil {
			t.Fatalf("CreateFile() failed: %s", err)
		}

		// create a symlink
		if err := target.CreateSymlink(testFile, testSymlink, false); err != nil {
			t.Fatalf("CreateSymlink() failed: %s", err)
		}

		// check if symlink exists
		lstat, err := target.Lstat(testSymlink)
		if err != nil {
			t.Fatalf("CreateSymlink() failed: %s", err)
		}
		if lstat.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("CreateSymlink() failed: %s", "not a symlink")
		}

		// create a symlink with overwrite
		if err := target.CreateSymlink(testSymlink, testFile, true); err != nil {
			t.Fatalf("CreateSymlink() with overwrite failed: %s", err)
		}

		// create a symlink with overwrite expect fail
		if err := target.CreateSymlink(testSymlink, testFile, false); err == nil {
			t.Fatalf("CreateSymlink() with disabled overwrite try to overwrite failed: %s", err)
		}
	}

}
