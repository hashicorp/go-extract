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

func TestUnpackBrotli(t *testing.T) {
	inputData := []byte("Hello, World!")

	tests := []struct {
		name         string
		archiveName  string
		expectedName string
		cfg          *config.Config
		generator    func(target string, data []byte) io.Reader
		testData     []byte
		wantErr      bool
	}{
		{
			name:         "Test unpack brotli",
			archiveName:  "test.br",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    newTestFile,
			testData:     compressBrotli(t, inputData),
			wantErr:      false,
		},
		{
			name:         "Test unpack brotli with no file extension",
			archiveName:  "test",
			expectedName: "test.decompressed",
			cfg:          config.NewConfig(),
			generator:    newTestFile,
			testData:     compressBrotli(t, inputData),
			wantErr:      false,
		},
		{
			name:         "Test unpack brotli read from buffer",
			expectedName: "goextract-decompressed-content",
			cfg:          config.NewConfig(),
			generator:    createByteReader,
			testData:     compressBrotli(t, inputData),
			wantErr:      false,
		},
		{
			name:         "Test unpack random bytes",
			archiveName:  "random",
			expectedName: "goextract-decompressed-content",
			cfg:          config.NewConfig(),
			generator:    newTestFile,
			testData:     inputData,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new target
			testingTarget := NewOS()

			// Create a temporary file (if necessary)
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.archiveName)

			// Generate the compressed file
			reader := tt.generator(tmpFile, tt.testData)
			if closer, ok := reader.(io.Closer); ok {
				defer closer.Close()
			}

			// Unpack the compressed file
			err := UnpackBrotli(context.Background(), testingTarget, tmpDir, reader, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("unpackBrotli() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {

				// Check if the file was created
				_, err = os.Stat(filepath.Join(tmpDir, tt.expectedName))
				if err != nil {
					t.Errorf("%s: Error checking if file was created: %v", tt.name, err)
				}

				// Check extracted file content
				data, err := os.ReadFile(filepath.Join(tmpDir, tt.expectedName))
				if err != nil {
					t.Errorf("%s: Error reading extracted file: %v", tt.name, err)
				}
				if string(data) != string(inputData) {
					t.Errorf("%v: Unpacked data is different from original data\n'%v'\n'%v'", tt.name, string(data), string(inputData))
				}

			}

		})
	}

}

func TestIsBrotli(t *testing.T) {
	header := []byte{0x00, 0x01, 0x02, 0x03} // replace with actual header bytes if needed

	if isBrotli(header) != false {
		t.Errorf("IsBrotli function failed, expected false, got true")
	}
}

func compressBrotli(t *testing.T, data []byte) []byte {
	brotliBuf := new(bytes.Buffer)
	brotliWriter := brotli.NewWriter(brotliBuf)

	_, err := brotliWriter.Write(data)
	if err != nil {
		t.Fatalf("error writing data to Brotli writer: %v", err)
	}

	err = brotliWriter.Close()
	if err != nil {
		t.Fatalf("error closing Brotli writer: %v", err)
	}

	return brotliBuf.Bytes()
}
