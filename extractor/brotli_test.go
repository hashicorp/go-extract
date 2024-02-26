package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/hashicorp/go-extract/config"
)

func TestIsBrotli(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Brotli header",
			header: []byte{0xce, 0xb2, 0xcf, 0x81},
			want:   true,
		},
		{
			name:   "Non-Brotli header",
			header: []byte{0x00, 0x00, 0x00, 0x00},
			want:   false,
		},
		{
			name:   "Other test data",
			header: []byte{0x1b, 0x00, 0x00, 0x00, 0x04, 0x22, 0x4f, 0x18, 0x64, 0x40, 0x46, 0x0e, 0x00, 0x00, 0x00, 0xff, 0xff},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBrotli(tt.header); got != tt.want {
				t.Errorf("IsBrotli() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnpackBrotli(t *testing.T) {

	tests := []struct {
		name         string
		archiveName  string
		expectedName string
		cfg          *config.Config
		generator    func(ctx context.Context, target string, data []byte) (io.Reader, error)
		testData     []byte
		wantErr      bool
	}{
		{
			name:         "Test unpack brotli",
			archiveName:  "test.br",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createBrotliCompressedFile,
			testData:     []byte("Hello, World!"),
			wantErr:      false,
		},
		{
			name:         "Test unpack brotli with no file extension",
			archiveName:  "test",
			expectedName: "test.decompressed-br",
			cfg:          config.NewConfig(),
			generator:    createBrotliCompressedFile,
			testData:     []byte("Hello, World!"),
			wantErr:      false,
		},
		{
			name:         "Test unpack brotli read from buffer",
			expectedName: "decompressed-br",
			cfg:          config.NewConfig(),
			generator:    createBrotliCompressedBuffer,
			testData:     []byte("Hello, World!"),
			wantErr:      false,
		},
		{
			name:         "Test unpack random bytes",
			archiveName:  "random",
			expectedName: "decompressed-br",
			cfg:          config.NewConfig(),
			generator:    writeBytesToFile,
			testData:     []byte("Strings and bytes and bytes and strings"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file (if necessary)
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.archiveName)

			// Generate the compressed file
			reader, err := tt.generator(context.Background(), tmpFile, tt.testData)
			if err != nil {
				t.Errorf("Error generating compressed file: %v", err)
			}

			// Unpack the compressed file
			err = UnpackBrotli(context.Background(), reader, tmpDir, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("unpackBrotli() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {

				// Check if the file was created
				_, err = os.Stat(filepath.Join(tmpDir, tt.expectedName))
				if err != nil {
					t.Errorf("Error checking if file was created: %v", err)
				}

				// Check extracted file content
				data, err := os.ReadFile(filepath.Join(tmpDir, tt.expectedName))
				if err != nil {
					t.Errorf("Error reading extracted file: %v", err)
				}
				if string(data) != string(tt.testData) {
					t.Errorf("Unpacked data is different from original data")
				}

			}

		})
	}

}

// Create a Brotli compressed file
func createBrotliCompressedFile(ctx context.Context, target string, data []byte) (io.Reader, error) {

	// Write the compressed data to the target file
	err := os.WriteFile(target, toBrotli(data), 0644)
	if err != nil {
		return nil, fmt.Errorf("Error writing compressed data to file: %v", err)
	}

	return os.Open(target)
}

// Create a Brotli compressed buffer
// ignore target and return a bytes.Reader
func createBrotliCompressedBuffer(ctx context.Context, target string, data []byte) (io.Reader, error) {
	return bytes.NewReader(toBrotli(data)), nil
}

// Write a byte slice to a file
func writeBytesToFile(ctx context.Context, target string, data []byte) (io.Reader, error) {
	// Write data to file
	err := os.WriteFile(target, data, 0644)
	if err != nil {
		return nil, fmt.Errorf("Error writing data to file: %v", err)
	}
	return os.Open(target)
}

// Compress a byte slice with Brotli
func toBrotli(data []byte) []byte {
	// Create a new Brotli writer
	brotliBuf := new(bytes.Buffer)
	brotliWriter := brotli.NewWriter(brotliBuf)

	// Write the data to the Brotli writer
	_, err := brotliWriter.Write(data)
	if err != nil {
		return nil
	}

	// Close the Brotli writer
	err = brotliWriter.Close()
	if err != nil {
		return nil
	}

	return brotliBuf.Bytes()
}
