package target

import (
	"bytes"
	"testing"
)

// TestCreateDir tests the CreateDir function from Mem
func TestMemCreateDir(t *testing.T) {

	// create a new ^target
	n := NewMemTarget()

	// check empty path
	if err := n.CreateDir("", 0755); err == nil {
		t.Fatalf("CreateDir() did not fail, but error was expected!")
	}

	// create file
	fName := "test_file"
	fReader := bytes.NewReader([]byte("test content"))
	if _, err := n.CreateFile(fName, fReader, 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// create folder with existing file in path
	if err := n.CreateDir(fName, 0755); err == nil {
		t.Fatalf("CreateDir() did not fail, but error was expected!")
	}

}

// TestLstat tests the Lstat function from Mem
func TestMemLstat(t *testing.T) {

	// create a new target
	n := NewMemTarget()

	// check empty path
	if _, err := n.Lstat(""); err == nil {
		t.Fatalf("Lstat() did not fail, but error was expected!")
	}

	// create file
	fName := "test_file"
	content := []byte("test content")
	fReader := bytes.NewReader(content)
	if _, err := n.CreateFile(fName, fReader, 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// check if file exists
	s, err := n.Lstat(fName)
	if err != nil {
		t.Fatalf("Lstat() failed: %s", err)
	}

	// check if file is a file
	if !s.Mode().IsRegular() {
		t.Fatalf("Lstat() failed: not a file")
	}

	if s.Name() != fName {
		t.Fatalf("Lstat() failed: wrong name")
	}

	if s.Size() != int64(len(content)) {
		t.Fatalf("Lstat() %v %v failed: wrong size", s.Size(), int64(len(content)))
	}

	if s.IsDir() {
		t.Fatalf("Lstat() failed: is a directory")
	}

	if _, err := n.Lstat("."); err != nil {
		t.Fatalf("Lstat() failed: dot stat")
	}

}
