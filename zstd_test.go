package extract_test

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
)

func TestIsZstd(t *testing.T) {
	// test cases
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0x28, 0xb5, 0x2f, 0xfd}, true},
		{[]byte{0x28, 0xb5, 0x2f, 0xfe}, false},
	}

	// run tests
	for _, test := range tests {
		if got := extract.IsZstd(test.header); got != test.want {
			t.Errorf("IsZstandard(%v) = %v, want %v", test.header, got, test.want)
		}
	}
}

func TestUnpackZstd(t *testing.T) {
	tests := []struct {
		name          string
		archiveName   string
		expectedName  string
		cfg           *extract.Config
		generator     func(target string, data []byte) io.Reader
		data          []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:         "TestUnpackZstandard",
			archiveName:  "test.zst",
			expectedName: "test",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			data:         compressZstd(t, []byte("test data")),
			wantErr:      false,
		},
		{
			name:          "TestUnpackZstandardCancelContext",
			archiveName:   "test.zst",
			expectedName:  "test",
			cfg:           extract.NewConfig(),
			generator:     newTestFile,
			data:          compressZstd(t, []byte("test data")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:         "TestUnpackZstandardInvalidData",
			archiveName:  "test.zst",
			expectedName: "test",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			data:         []byte("test data"),
			wantErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a context
			ctx := context.Background()

			// Create a new target
			testingTarget := extract.NewDisk()

			// Create a temporary directory
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, test.archiveName)

			// Create the source file
			src := test.generator(archivePath, test.data)
			if closer, ok := src.(io.Closer); ok {
				defer closer.Close()
			}

			// Create a context
			if test.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// unpack
			if err := extract.UnpackZstd(ctx, testingTarget, tmpDir, src, test.cfg); (err != nil) != test.wantErr {
				t.Errorf("UnpackZstandard() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}

}
