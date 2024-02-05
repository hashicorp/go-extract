package extractor

import (
	"strings"
	"testing"
)

func TestNewHeaderReader(t *testing.T) {
	reader := strings.NewReader("test input")
	headerReader, err := NewHeaderReader(reader, 4)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if string(headerReader.PeekHeader()) != "test" {
		t.Errorf("Incorrect header: got %v, want %v", string(headerReader.PeekHeader()), "test")
	}

	buf := make([]byte, 4)
	n, err := headerReader.Read(buf)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != 4 {
		t.Errorf("Incorrect number of bytes read: got %v, want %v", n, 4)
	}

	if string(buf) != "test" {
		t.Errorf("Incorrect data read: got %v, want %v", string(buf), "test")
	}

	n, err = headerReader.Read(buf)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != 4 {
		t.Errorf("Incorrect number of bytes read: got %v, want %v", n, 4)
	}

	if string(buf) != " inp" {
		t.Errorf("Incorrect data read: got %v, want %v", string(buf), " inp")
	}
}
