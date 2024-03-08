package extractor

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

func TestIsZlib(t *testing.T) {
	// test cases
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0x78, 0x01}, true},
		{[]byte{0x00, 0x00}, false},
	}

	// run tests
	for _, tt := range tests {
		if got := IsZlib(tt.header); got != tt.want {
			t.Errorf("IsZlib(%v) = %v; want %v", tt.header, got, tt.want)
		}
	}
}

func TestUnpackZlib(t *testing.T) {

	tests := []struct {
		name          string
		cfg           *config.Config
		generator     func(target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:      "UnpackZlib",
			cfg:       config.NewConfig(),
			generator: createFile,
			testData:  compressZlib([]byte("test")),
			wantErr:   false,
		},
		{
			name:          "UnpackZlib with cancel",
			cfg:           config.NewConfig(),
			generator:     createFile,
			testData:      compressZlib([]byte("test")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:      "UnpackZlib with limited input",
			cfg:       config.NewConfig(config.WithMaxInputSize(1)),
			generator: createFile,
			testData:  compressZlib([]byte("test")),
			wantErr:   true,
		},
		{
			name:      "UnpackZlib with invalid input",
			cfg:       config.NewConfig(),
			generator: createFile,
			testData:  []byte("this is not valid zlib data"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// prepare context
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// create a temporary file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.zz")

			// create a reader
			src := tt.generator(testFile, tt.testData)

			// cancel if necessary
			if tt.cancelContext {
				cancel()
			}

			// run the test
			err := UnpackZlib(ctx, src, tmpDir, tt.cfg)
			if (err != nil) != tt.wantErr {
				data, _ := os.ReadFile("test")
				t.Errorf("UnpackZlib() error = %v, wantErr %v\n'%v'", err, tt.wantErr, string(data))
				return
			}
		})
	}
}

// compressZlib compresses the data using the zlib algorithm
func compressZlib(data []byte) []byte {

	// Create a new zlib writer
	var buf bytes.Buffer

	w := zlib.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		panic(fmt.Errorf("error writing data to zlib writer: %w", err))
	}
	err = w.Close()
	if err != nil {
		panic(fmt.Errorf("error closing zlib writer: %w", err))
	}

	return buf.Bytes()
}
