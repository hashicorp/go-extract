package extractor

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/pierrec/lz4/v4"
)

func TestIsLZ4(t *testing.T) {
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0x04, 0x22, 0x4D, 0x18}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00}, false},
	}

	for _, test := range tests {
		if got := isLZ4(test.header); got != test.want {
			t.Errorf("IsLZ4(%v) = %v; want %v", test.header, got, test.want)
		}
	}

}

func TestUnpackLZ4(t *testing.T) {
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
			testData:  compressLZ4(t, []byte("test")),
			wantErr:   false,
		},
		{
			name:          "UnpackLZ4 with cancel",
			cfg:           config.NewConfig(),
			generator:     newTestFile,
			testData:      compressLZ4(t, []byte("test")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:      "UnpackLZ4 with limited input",
			cfg:       config.NewConfig(config.WithMaxInputSize(1)),
			generator: newTestFile,
			testData:  compressLZ4(t, []byte("test")),
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new target
			testingTarget := NewOS()

			// prepare context
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// create a temporary file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.lz4")

			// create a reader
			src := test.generator(testFile, test.testData)
			defer func() {
				if closer, ok := src.(io.Closer); ok {
					closer.Close()
				}
			}()

			// cancel if necessary
			if test.cancelContext {
				cancel()
			}

			// run the test
			err := UnpackLZ4(ctx, testingTarget, tmpDir, src, test.cfg)
			if (err != nil) != test.wantErr {
				data, _ := os.ReadFile("test")
				t.Errorf("%v UnpackLZ4() error = %v, wantErr %v\n'%v'", test.name, err, test.wantErr, string(data))
				return
			}
		})
	}
}

func compressLZ4(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)

	_, err := w.Write(data)
	if err != nil {
		t.Fatalf("error writing data to lz4 writer: %v", err)
	}

	err = w.Close()
	if err != nil {
		t.Fatalf("error closing lz4 writer: %v", err)
	}

	return buf.Bytes()
}
