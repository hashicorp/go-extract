package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/pierrec/lz4/v4"
)

func TestIsLZ4(t *testing.T) {

	// test cases
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0x04, 0x22, 0x4D, 0x18}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00}, false},
	}

	// run tests
	for _, tt := range tests {
		if got := IsLZ4(tt.header); got != tt.want {
			t.Errorf("IsLZ4(%v) = %v; want %v", tt.header, got, tt.want)
		}
	}

}

func TestUnpackLZ4(t *testing.T) {

	// test cases
	tests := []struct {
		name          string
		cfg           *config.Config
		generator     func(target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:      "UnpackLZ4",
			cfg:       config.NewConfig(),
			generator: newTestFile,
			testData:  compressLZ4([]byte("test")),
			wantErr:   false,
		},
		{
			name:          "UnpackLZ4 with cancel",
			cfg:           config.NewConfig(),
			generator:     newTestFile,
			testData:      compressLZ4([]byte("test")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:      "UnpackLZ4 with limited input",
			cfg:       config.NewConfig(config.WithMaxInputSize(1)),
			generator: newTestFile,
			testData:  compressLZ4([]byte("test")),
			wantErr:   true,
		},
		{
			name:      "UnpackLZ4 with invalid input",
			cfg:       config.NewConfig(),
			generator: newTestFile,
			testData:  []byte("this is not valid zlib data"),
			wantErr:   true,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// prepare context
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// create a temporary file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.lz4")

			// create a reader
			src := tt.generator(testFile, tt.testData)
			defer func() {
				if closer, ok := src.(io.Closer); ok {
					closer.Close()
				}
			}()

			// cancel if necessary
			if tt.cancelContext {
				cancel()
			}

			// run the test
			err := UnpackLZ4(ctx, testingTarget, tmpDir, src, tt.cfg)
			if (err != nil) != tt.wantErr {
				data, _ := os.ReadFile("test")
				t.Errorf("%v UnpackLZ4() error = %v, wantErr %v\n'%v'", tt.name, err, tt.wantErr, string(data))
				return
			}
		})
	}
}

// compressLZ4 compresses data using the LZ4 algorithm
func compressLZ4(data []byte) []byte {

	// Create a new lz4 writer
	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)

	// Write the data to the lz4 writer
	_, err := w.Write(data)
	if err != nil {
		panic(fmt.Errorf("error writing data to lz4 writer: %w", err))
	}
	err = w.Close()
	if err != nil {
		panic(fmt.Errorf("error closing lz4 writer: %w", err))
	}

	// Return the compressed data
	return buf.Bytes()
}
