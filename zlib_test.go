package extract_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
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
		if got := extract.IsZlib(tt.header); got != tt.want {
			t.Errorf("IsZlib(%v) = %v; want %v", tt.header, got, tt.want)
		}
	}
}

func TestUnpackZlib(t *testing.T) {

	tests := []struct {
		name          string
		cfg           *extract.Config
		generator     func(target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:      "UnpackZlib",
			cfg:       extract.NewConfig(),
			generator: newTestFile,
			testData:  compressZlib(t, []byte("test")),
			wantErr:   false,
		},
		{
			name:          "UnpackZlib with cancel",
			cfg:           extract.NewConfig(),
			generator:     newTestFile,
			testData:      compressZlib(t, []byte("test")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:      "UnpackZlib with limited input",
			cfg:       extract.NewConfig(extract.WithMaxInputSize(1)),
			generator: newTestFile,
			testData:  compressZlib(t, []byte("test")),
			wantErr:   true,
		},
		{
			name:      "UnpackZlib with invalid input",
			cfg:       extract.NewConfig(),
			generator: newTestFile,
			testData:  []byte("this is not valid zlib data"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// prepare context
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create a new target
			testingTarget := extract.NewDisk()

			// create a temporary file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.zz")

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
			err := extract.UnpackZlib(ctx, testingTarget, tmpDir, src, tt.cfg)
			if (err != nil) != tt.wantErr {
				data, _ := os.ReadFile("test")
				t.Errorf("UnpackZlib() error = %v, wantErr %v\n'%v'", err, tt.wantErr, string(data))
				return
			}
		})
	}
}
