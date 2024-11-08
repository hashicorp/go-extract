package extract

import (
	"bytes"
	"io"
	"testing"
)

func Test_getHeader(t *testing.T) {
	tests := []struct {
		name    string
		src     io.Reader
		wantErr bool
	}{
		{
			name:    "Read header from bytes.Buffer (implements io.Seeker)",
			src:     bytes.NewBuffer([]byte("test data")),
			wantErr: false,
		},
		{
			name:    "Read header from bytes.Reader (implements io.Seeker)",
			src:     bytes.NewReader([]byte("test data")),
			wantErr: false,
		},
		// Add more test cases as needed
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := GetHeader(test.src)
			if (err != nil) != test.wantErr {
				t.Errorf("getHeader() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

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
	link, err := mem.readlink(testLink)
	if err != nil {
		t.Fatalf("Readlink() failed: %s", err)
	}

	if link != testPath {
		t.Fatalf("Readlink() failed: expected %s, got %s", testPath, link)
	}

	// read a symlink that does not exist
	if _, err := mem.readlink(testPathNotExist); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}

	// create a file
	if _, err := mem.CreateFile(testPath, bytes.NewReader([]byte("test")), 0644, false, -1); err != nil {
		t.Fatalf("CreateFile() failed: %s", err)
	}

	// readlink a file
	if _, err := mem.readlink(testPath); err == nil {
		t.Fatalf("Readlink() failed: expected error, got nil")
	}
}
