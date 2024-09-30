package target

import (
	"bytes"
	"io"
	"testing"
)

// TestLimitErrorWriter tests the LimitErrorWriter
func TestLimitErrorWriter(t *testing.T) {
	var buf bytes.Buffer
	l := newLimitErrorWriter(&buf, 5)

	n, err := l.Write([]byte("hello"))
	if n != 5 || err != nil {
		t.Errorf("Expected to write 5 bytes, but wrote %d with error %v", n, err)
	}

	n, err = l.Write([]byte("world"))
	if n != 0 || err != io.ErrShortWrite {
		t.Errorf("Expected to write 0 bytes and get ErrShortWrite, but wrote %d with error %v", n, err)
	}

	if buf.String() != "hello" {
		t.Errorf("Expected buffer to contain 'hello', but it contains '%s'", buf.String())
	}
}
