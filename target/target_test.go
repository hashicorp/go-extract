package target

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type TargetFunc func() Target

func testTargets(t *testing.T) []struct {
	name   string
	path   string
	link   string
	file   string
	data   []byte
	target Target
} {
	tmpDir := t.TempDir()
	testData := []byte("test data")
	return []struct {
		name   string
		path   string
		link   string
		file   string
		data   []byte
		target Target
	}{
		{
			name:   "os",
			path:   filepath.Join(tmpDir, "test"),
			link:   filepath.Join(tmpDir, "symlink"),
			file:   filepath.Join(tmpDir, "file"),
			data:   testData,
			target: NewOS(),
		},
		{
			name:   "Memory",
			path:   "test",
			link:   "symlink",
			file:   "file",
			data:   testData,
			target: NewMemory(),
		},
	}
}

// TestCreateDir tests the CreateDir function from Os
func TestCreateDir(t *testing.T) {

	for _, test := range testTargets(t) {
		t.Run(test.name, func(t *testing.T) {
			// Create a directory, expect success.
			if err := test.target.CreateDir(test.path, 0755); err != nil {
				t.Fatal(err)
			}

			// Check if directory exists, expect success.
			if _, err := test.target.Lstat(test.path); err != nil {
				t.Fatal(err)
			}

			// Create a directory that already exists, expect success.
			// This is a no-op, so it should not return an error.
			if err := test.target.CreateDir(test.path, 0755); err != nil {
				t.Fatal(err)
			}

			// create a file in the directory
			if _, err := test.target.CreateFile(test.file, bytes.NewReader(test.data), 0644, false, -1); err != nil {
				t.Fatalf("CreateFile() failed: %s", err)
			}

			// create a directory where a file already exists, expect fail
			if err := test.target.CreateDir(test.file, 0755); err == nil {
				t.Fatalf("CreateDir() succeeded, but error was expected")
			}
		})
	}
}

// TestCreateFile tests the CreateFile function from Os
func TestCreateFile(t *testing.T) {

	for _, test := range testTargets(t) {
		t.Run(test.name, func(t *testing.T) {

			// create a file
			if _, err := test.target.CreateFile(test.path, bytes.NewReader(test.data), 0644, false, -1); err != nil {
				t.Fatalf("CreateFile() failed: %s", err)
			}

			// check if file exists
			if _, err := test.target.Lstat(test.path); err != nil {
				t.Fatalf("Lstat() returned an error, but no error was expected: %s", err)
			}

			// overwrite the file
			if _, err := test.target.CreateFile(test.path, bytes.NewReader(test.data), 0644, true, -1); err != nil {
				t.Fatalf("overwriting file with CreateFile() failed with an error, but no error expected: %s", err)
			}

			// create a file with overwrite expect fail
			if _, err := test.target.CreateFile(test.path, bytes.NewReader(test.data), 0644, false, -1); err == nil {
				t.Fatalf("file overwrite succeeded, but error expected")
			}

			// create a file with maxSize
			maxSize := int64(5)
			if n, err := test.target.CreateFile(test.path, bytes.NewReader(test.data), 0644, true, maxSize); err == nil {
				t.Fatalf("file was created, but error was expected due to maxSize exceeded: err: %s, maxSize: %v n: %v", err, maxSize, n)
			}
		})
	}
}

// TestCreateSymlink tests the CreateSymlink function from Os
func TestCreateSymlink(t *testing.T) {

	for _, test := range testTargets(t) {
		t.Run(test.name, func(t *testing.T) {

			// create a file
			if _, err := test.target.CreateFile(test.path, bytes.NewReader(test.data), 0644, false, -1); err != nil {
				t.Fatalf("CreateFile() failed with an error, but no error was expected: %s", err)
			}

			// create a symlink
			if err := test.target.CreateSymlink(test.path, test.link, false); err != nil {
				t.Fatalf("CreateSymlink() failed with an error, but no error was expected: %s", err)
			}

			// check if symlink exists
			lstat, err := test.target.Lstat(test.link)
			if err != nil {
				t.Fatalf("Lstat() returned an error, but no error was expected: %s", err)
			}
			if lstat.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("CreateSymlink() failed: %s", "not a symlink")
			}

			// create a symlink with overwrite
			if err := test.target.CreateSymlink(test.link, test.path, true); err != nil {
				t.Fatalf("CreateSymlink() with overwrite failed, but no error was expected: %s", err)
			}

			// create a symlink with overwrite expect fail
			if err := test.target.CreateSymlink(test.link, test.path, false); err == nil {
				t.Fatalf("CreateSymlink() with disabled overwrite try to let the function fail, but error returned: %s", err)
			}

		})
	}
}
