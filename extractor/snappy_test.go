package extractor

import (
	"bytes"
	"context"
	"fmt"
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
	for _, tt := range tests {
		if got := IsSnappy(tt.header); got != tt.want {
			t.Errorf("IsSnappy(%v) = %v, want %v", tt.header, got, tt.want)
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
			generator:     createFile,
			testData:      compressSnappy([]byte("test data")),
			cancelContext: false,
			wantErr:       false,
		},
		{
			name:          "Test snappy unpacking with canceled context",
			archiveName:   "test.sz",
			expectedName:  "test",
			cfg:           config.NewConfig(),
			generator:     createFile,
			testData:      compressSnappy([]byte("test data")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:          "Test snappy unpacking with invalid file",
			archiveName:   "test.sz",
			expectedName:  "test",
			cfg:           config.NewConfig(),
			generator:     createFile,
			testData:      []byte("test data"),
			cancelContext: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create a context
			ctx := context.Background()

			// Create a temporary directory
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, tt.archiveName)

			// Create the source file
			src := tt.generator(archivePath, tt.testData)
			if closer, ok := src.(io.Closer); ok {
				defer closer.Close()
			}

			// Create a context
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// Unpack the source file
			if err := UnpackSnappy(ctx, src, tmpDir, tt.cfg); (err != nil) != tt.wantErr {
				t.Errorf("UnpackSnappy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

// compressSnappy compresses the data using the snappy algorithm
func compressSnappy(data []byte) []byte {
	// Create a new snappy writer
	var buf bytes.Buffer
	w := snappy.NewBufferedWriter(&buf)

	_, err := w.Write(data)
	if err != nil {
		panic(fmt.Errorf("error writing data to snappy writer: %w", err))
	}
	if err := w.Close(); err != nil {
		panic(fmt.Errorf("error closing snappy writer: %w", err))
	}
	return buf.Bytes()
}
