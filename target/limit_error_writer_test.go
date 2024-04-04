package target

import (
	"bytes"
	"io"
	"testing"
)

// TestLimitErrorWriter tests the LimitErrorWriter
func TestLimitErrorWriter(t *testing.T) {
	var buf bytes.Buffer
	l := NewLimitErrorWriter(&buf, 8)

	n, err := l.Write([]byte("Hello"))
	if n != 5 || err != nil {
		t.Errorf("Expected to write 5 bytes, but wrote %d with error %v", n, err)
	}

	n, err = l.Write([]byte("World"))
	if n != 3 || err != io.ErrShortWrite {
		t.Errorf("Expected to write 3 bytes and get ErrShortWrite, but wrote %d with error %v", n, err)
	}

	n, err = l.Write([]byte("world"))
	if n != 0 || err != io.ErrShortWrite {
		t.Errorf("Expected to write 0 bytes and get ErrShortWrite, but wrote %d with error %v", n, err)
	}

	if buf.String() != "HelloWor" {
		t.Errorf("Expected buffer to contain 'HelloWor', but it contains '%s'", buf.String())
	}
}
