package extractor

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/golang/snappy"
	"github.com/hashicorp/go-extract/config"
)

// TestIsSnappy checks if the header matches the snappy magic bytes.
func TestIsSnappy(t *testing.T) {
	// test cases
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0xff, 0x06, 0x00, 0x00, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59}, true},
		{[]byte{0xff, 0x06, 0x00, 0x00, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x00}, false},
	}

	// run tests
	for _, test := range tests {
		if got := isSnappy(test.header); got != test.want {
			t.Errorf("IsSnappy(%v) = %v, want %v", test.header, got, test.want)
		}
	}
}

func TestUnpackSnappy(t *testing.T) {
	tests := []struct {
		name          string
		archiveName   string
		expectedName  string
		cfg           *config.Config
		generator     func(target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:          "Test snappy unpacking",
			archiveName:   "test.sz",
			expectedName:  "test",
			cfg:           config.NewConfig(),
			generator:     newTestFile,
			testData:      compressSnappy(t, []byte("test data")),
			cancelContext: false,
			wantErr:       false,
		},
		{
			name:          "Test snappy unpacking with canceled context",
			archiveName:   "test.sz",
			expectedName:  "test",
			cfg:           config.NewConfig(),
			generator:     newTestFile,
			testData:      compressSnappy(t, []byte("test data")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:          "Test snappy unpacking with invalid file",
			archiveName:   "test.sz",
			expectedName:  "test",
			cfg:           config.NewConfig(),
			generator:     newTestFile,
			testData:      []byte("test data"),
			cancelContext: false,
			wantErr:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new target
			testingTarget := NewOS()

			// Create a context
			ctx := context.Background()

			// Create a temporary directory
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, test.archiveName)

			// Create the source file
			src := test.generator(archivePath, test.testData)
			if closer, ok := src.(io.Closer); ok {
				defer closer.Close()
			}

			// Create a context
			if test.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// Unpack the source file
			if err := UnpackSnappy(ctx, testingTarget, tmpDir, src, test.cfg); (err != nil) != test.wantErr {
				t.Errorf("UnpackSnappy() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}

}

// compressSnappy compresses the data using the snappy algorithm
func compressSnappy(t *testing.T, data []byte) []byte {
	t.Helper()

	// Create a new snappy writer
	var buf bytes.Buffer
	w := snappy.NewBufferedWriter(&buf)

	_, err := w.Write(data)
	if err != nil {
		t.Fatalf("error writing data to snappy writer: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("error closing snappy writer: %v", err)
	}
	return buf.Bytes()
}