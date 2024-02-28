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
	"github.com/klauspost/compress/zstd"
)

// TestIsZstd tests the IsZstd function.
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
	for _, tt := range tests {
		if got := IsZstd(tt.header); got != tt.want {
			t.Errorf("IsZstandard(%v) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

// TestUnpackZstd tests the UnpackZstd function.
func TestUnpackZstd(t *testing.T) {
	tests := []struct {
		name          string
		archiveName   string
		expectedName  string
		cfg           *config.Config
		generator     func(ctx context.Context, target string, data []byte) io.Reader
		testData      []byte
		cancelContext bool
		wantErr       bool
	}{
		{
			name:         "TestUnpackZstandard",
			archiveName:  "test.zst",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData:     compressZstd([]byte("test data")),
			wantErr:      false,
		},
		{
			name:          "TestUnpackZstandardCancelContext",
			archiveName:   "test.zst",
			expectedName:  "test",
			cfg:           config.NewConfig(),
			generator:     createFile,
			testData:      compressZstd([]byte("test data")),
			cancelContext: true,
			wantErr:       true,
		},
		{
			name:         "TestUnpackZstandardInvalidData",
			archiveName:  "test.zst",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData:     []byte("test data"),
			wantErr:      true,
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
			src := tt.generator(ctx, archivePath, tt.testData)
			if closer, ok := src.(io.Closer); ok {
				defer closer.Close()
			}

			// Create a context
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// unpack
			if err := UnpackZstd(ctx, src, tmpDir, tt.cfg); (err != nil) != tt.wantErr {
				t.Errorf("UnpackZstandard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

// createFile creates file with byte content
func createFile(ctx context.Context, target string, data []byte) io.Reader {

	// Write the compressed data to the file
	if err := os.WriteFile(target, data, 0644); err != nil {
		panic(fmt.Errorf("error writing compressed data to file: %w", err))
	}

	// Open the file
	if f, err := os.Open(target); err != nil {
		panic(fmt.Errorf("error stating file: %w", err))
	} else {
		return f
	}
}

// compressZstd compresses the data using the zstandard algorithm
func compressZstd(data []byte) []byte {

	// Create a new zst writer
	var buf bytes.Buffer

	enc, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		panic(fmt.Errorf("error creating zstd writer: %w", err))
	}

	_, err = enc.Write(data)
	enc.Close()
	if err != nil {
		panic(fmt.Errorf("error writing data to zstd writer: %w", err))
	}

	return buf.Bytes()
}
