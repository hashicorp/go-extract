package extract_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
)

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
			if got := extract.IsBzip2(tt.header); got != tt.want {
				t.Errorf("IsBzip2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnpackBzip2(t *testing.T) {
	testData := []byte("Hello, World!")

	tests := []struct {
		name         string
		testFileName string
		expectedName string
		cfg          *extract.Config
		generator    func(target string, data []byte) io.Reader
		testData     []byte
	}{
		{
			name:         "Test unpack bzip2",
			testFileName: "test.bz2",
			expectedName: "test",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			testData:     compressBzip2(t, testData),
		},
		{
			name:         "Test unpack bzip2 with no file extension",
			testFileName: "test",
			expectedName: "test.decompressed",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			testData:     compressBzip2(t, testData),
		},
		{
			name:         "Test unpack bzip2 read from buffer",
			expectedName: "goextract-decompressed-content",
			cfg:          extract.NewConfig(),
			generator:    createByteReader,
			testData:     compressBzip2(t, testData),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create a new target
			testingTarget := extract.NewDisk()

			// create a temporary file
			tmpDir := t.TempDir()
			tFile := filepath.Join(tmpDir, tt.testFileName)

			// generate the file
			src := tt.generator(tFile, tt.testData)
			if closer, ok := src.(io.Closer); ok {
				defer closer.Close()
			}

			// Unpack the file
			err := extract.UnpackBzip2(context.Background(), testingTarget, tmpDir, src, tt.cfg)
			if err != nil {
				t.Errorf("%v: UnpackBzip2() error = %v", tt.name, err)
				return
			}

			// Check extracted file content
			data, err := os.ReadFile(filepath.Join(tmpDir, tt.expectedName))
			if err != nil {
				t.Errorf("%v: Error reading extracted file: %v", tt.name, err)
			}
			if string(data) != string(testData) {
				t.Errorf("Unpacked data is different from original data\n%v\n%v", string(data), string(tt.testData))
			}

		})
	}

}
