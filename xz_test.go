package extract_test

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
)

func Test_isXz(t *testing.T) {
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, false},
	}

	for _, tt := range tests {
		if got := extract.IsXz(tt.header); got != tt.want {
			t.Errorf("IsXz(%v) = %v; want %v", tt.header, got, tt.want)
		}
	}
}

func TestUnpackXz(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *extract.Config
		generator     func(target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:      "UnpackXz",
			cfg:       extract.NewConfig(),
			generator: newTestFile,
			testData:  compressXz(t, []byte("test")),
			wantErr:   false,
		},
		{
			name:          "UnpackXz with cancel",
			cfg:           extract.NewConfig(),
			generator:     newTestFile,
			testData:      compressXz(t, []byte("test")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:      "UnpackXz with limited input",
			cfg:       extract.NewConfig(extract.WithMaxInputSize(1)),
			generator: newTestFile,
			testData:  compressXz(t, []byte("test")),
			wantErr:   true,
		},
		{
			name:      "UnpackXz with invalid input",
			cfg:       extract.NewConfig(),
			generator: newTestFile,
			testData:  []byte("this is not valid xz data"),
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, "test.xz")

			// Create a new target
			testingTarget := extract.NewDisk()

			// create a temporary file
			reader := test.generator(archivePath, test.testData)
			if closer, ok := reader.(io.Closer); ok {
				defer closer.Close()
			}

			// cancel the context
			if test.cancelContext {
				cancel()
			}

			// unpack the file
			err := extract.UnpackXz(ctx, testingTarget, tmpDir, reader, test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("UnpackXz() error = %v; wantErr %v", err, test.wantErr)
			}
		})
	}
}
