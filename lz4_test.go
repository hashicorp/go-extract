package extract_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
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
		if got := extract.IsLZ4(test.header); got != test.want {
			t.Errorf("IsLZ4(%v) = %v; want %v", test.header, got, test.want)
		}
	}

}

func TestUnpackLZ4(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *extract.Config
		generator     func(target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:      "UnpackLZ4",
			cfg:       extract.NewConfig(),
			generator: newTestFile,
			testData:  compressLZ4(t, []byte("test")),
			wantErr:   false,
		},
		{
			name:          "UnpackLZ4 with cancel",
			cfg:           extract.NewConfig(),
			generator:     newTestFile,
			testData:      compressLZ4(t, []byte("test")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:      "UnpackLZ4 with limited input",
			cfg:       extract.NewConfig(extract.WithMaxInputSize(1)),
			generator: newTestFile,
			testData:  compressLZ4(t, []byte("test")),
			wantErr:   true,
		},
		{
			name:      "UnpackLZ4 with invalid input",
			cfg:       extract.NewConfig(),
			generator: newTestFile,
			testData:  []byte("this is not valid zlib data"),
			wantErr:   true,
		},
	}

	// run tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new target
			testingTarget := extract.NewDisk()

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
			err := extract.UnpackLZ4(ctx, testingTarget, tmpDir, src, test.cfg)
			if (err != nil) != test.wantErr {
				data, _ := os.ReadFile("test")
				t.Errorf("%v UnpackLZ4() error = %v, wantErr %v\n'%v'", test.name, err, test.wantErr, string(data))
				return
			}
		})
	}
}
