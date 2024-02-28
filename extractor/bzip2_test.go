package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/dsnet/compress/bzip2"
	"github.com/hashicorp/go-extract/config"
)

// TestIsBzip2 tests the IsBzip2 function.
func TestIsBzip2(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "BZh1",
			header: []byte("BZh1"),
			want:   true,
		},
		{
			name:   "BZh9",
			header: []byte("BZh9"),
			want:   true,
		},
		{
			name:   "Not Bzip2",
			header: []byte("Not Bzip2"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBzip2(tt.header); got != tt.want {
				t.Errorf("IsBzip2() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUnpackBzip2 tests the UnpackBzip2 function.
func TestUnpackBzip2(t *testing.T) {
	tests := []struct {
		name         string
		testFileName string
		expectedName string
		cfg          *config.Config
		generator    func(ctx context.Context, target string, data []byte) io.Reader
		testData     []byte
		wantErr      bool
	}{
		{
			name:         "Test unpack bzip2",
			testFileName: "test.bz2",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator: func(ctx context.Context, target string, data []byte) io.Reader {
				return createFile(ctx, target, compressBzip2(data))
			},
			testData: []byte("test data"),
			wantErr:  false,
		},
		{
			name:         "Test unpack bzip2 with no file extension",
			testFileName: "test",
			expectedName: "test.decompressed-bz2",
			cfg:          config.NewConfig(),
			generator: func(ctx context.Context, target string, data []byte) io.Reader {
				return createFile(ctx, target, compressBzip2(data))
			},
			testData: []byte("test data"),
			wantErr:  false,
		},
		{
			name:         "Test unpack bzip2 read from buffer",
			expectedName: "decompressed-bz2",
			cfg:          config.NewConfig(),
			generator: func(ctx context.Context, target string, data []byte) io.Reader {
				return bytes.NewReader(compressBzip2(data))
			},
			testData: []byte("test data"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create a temporary file
			tmpDir := t.TempDir()
			target := filepath.Join(tmpDir, tt.testFileName)

			// generate the file
			src := tt.generator(context.Background(), target, tt.testData)
			if closer, ok := src.(io.Closer); ok {
				defer closer.Close()
			}

			// Unpack the file
			err := UnpackBzip2(context.Background(), src, tmpDir, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnpackBzip2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}

}

// compressBzip2 compresses data with bzip2 algorithm.
func compressBzip2(data []byte) []byte {
	// Create a new Bzip2 writer
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, &bzip2.WriterConfig{
		Level: bzip2.DefaultCompression,
	})
	if err != nil {
		panic(fmt.Errorf("error creating bzip2 writer: %w", err))
	}

	// Write the data to the Bzip2 writer
	_, err = w.Write(data)
	if err != nil {
		panic(fmt.Errorf("error writing data to bzip2 writer: %w", err))
	}

	// Close the Bzip2 writer
	err = w.Close()
	if err != nil {
		panic(fmt.Errorf("error closing bzip2 writer: %w", err))
	}

	return buf.Bytes()
}
