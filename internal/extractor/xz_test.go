package extractor

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/ulikunitz/xz"
)

func Test_isXz(t *testing.T) {
	// test cases
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, false},
	}

	// run tests
	for _, tt := range tests {
		if got := isXz(tt.header); got != tt.want {
			t.Errorf("IsXz(%v) = %v; want %v", tt.header, got, tt.want)
		}
	}
}

func TestUnpackXz(t *testing.T) {

	tests := []struct {
		name          string
		cfg           *config.Config
		generator     func(target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:      "UnpackXz",
			cfg:       config.NewConfig(),
			generator: newTestFile,
			testData:  compressXz(t, []byte("test")),
			wantErr:   false,
		},
		{
			name:          "UnpackXz with cancel",
			cfg:           config.NewConfig(),
			generator:     newTestFile,
			testData:      compressXz(t, []byte("test")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:      "UnpackXz with limited input",
			cfg:       config.NewConfig(config.WithMaxInputSize(1)),
			generator: newTestFile,
			testData:  compressXz(t, []byte("test")),
			wantErr:   true,
		},
		{
			name:      "UnpackXz with invalid input",
			cfg:       config.NewConfig(),
			generator: newTestFile,
			testData:  []byte("this is not valid xz data"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, "test.xz")

			// Create a new target
			testingTarget := NewOS()

			// create a temporary file
			reader := tt.generator(archivePath, tt.testData)
			if closer, ok := reader.(io.Closer); ok {
				defer closer.Close()
			}

			// cancel the context
			if tt.cancelContext {
				cancel()
			}

			// unpack the file
			err := UnpackXz(ctx, testingTarget, tmpDir, reader, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnpackXz() error = %v; wantErr %v", err, tt.wantErr)
			}
		})
	}

}

// compressXz compresses the data using the Xz algorithm
func compressXz(t *testing.T, data []byte) []byte {
	// Create a new xz writer
	var buf bytes.Buffer

	w, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatalf("error creating xz writer: %v", err)
	}

	_, err = w.Write(data)
	if err != nil {
		t.Fatalf("error writing data to xz writer: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("error closing xz writer: %v", err)
	}

	return buf.Bytes()
}
