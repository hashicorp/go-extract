package extractor

import (
	"bytes"
	"context"
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
		generator    func(ctx context.Context, target string, data []byte) io.Reader
		testData     []byte
		wantErr      bool
	}{
		{
			name:         "Test unpack brotli",
			archiveName:  "test.br",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData:     compressBrotli([]byte("Hello, World!")),
			wantErr:      false,
		},
		{
			name:         "Test unpack brotli with no file extension",
			archiveName:  "test",
			expectedName: "test.decompressed-br",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData:     compressBrotli([]byte("Hello, World!")),
			wantErr:      false,
		},
		{
			name:         "Test unpack brotli read from buffer",
			expectedName: "decompressed-br",
			cfg:          config.NewConfig(),
			generator:    createByteReader,
			testData:     []byte("Hello, World!"),
			wantErr:      false,
		},
		{
			name:         "Test unpack random bytes",
			archiveName:  "random",
			expectedName: "decompressed-br",
			cfg:          config.NewConfig(),
			generator:    createFile,
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
			reader := tt.generator(context.Background(), tmpFile, tt.testData)
			if closer, ok := reader.(io.Closer); ok {
				defer closer.Close()
			}

			// Unpack the compressed file
			err := UnpackBrotli(context.Background(), reader, tmpDir, tt.cfg)
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

// Compress a byte slice with Brotli
func compressBrotli(data []byte) []byte {
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
