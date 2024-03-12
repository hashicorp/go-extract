package extractor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

// TestIsGzip test with various test cases the implementation of IsGzip
func TestIsGZip(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Valid GZIP header",
			header: []byte{0x1f, 0x8b, 0x08},
			want:   true,
		},
		{
			name:   "Invalid GZIP header",
			header: []byte{0x1f, 0x7b, 0x07},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGZip(tt.header); got != tt.want {
				t.Errorf("IsGZIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGzipUnpack test with various test cases the implementation of zip.Unpack
func TestGzipUnpack(t *testing.T) {

	testData := []byte("Hello, World!")

	tests := []struct {
		name            string
		archiveName     string
		expectedName    string
		cfg             *config.Config
		generator       func(target string, data []byte) io.Reader
		testData        []byte
		contextCanceled bool
		wantErr         bool
	}{
		{
			name:         "normal gzip with file",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData:     compressGzip(testData),
			wantErr:      false,
		},
		{
			name:         "random file with no gzip",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData:     testData,
			wantErr:      true,
		},
		{
			name:         "gzip error while reading the header",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData:     []byte("123"),
			wantErr:      true,
		},
		{
			name:            "gzip with canceled context",
			archiveName:     "test.gz",
			expectedName:    "test",
			cfg:             config.NewConfig(),
			generator:       createFile,
			testData:        compressGzip(testData),
			contextCanceled: true,
			wantErr:         true,
		},
		{
			name:         "gzip with limited reader",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          config.NewConfig(config.WithMaxInputSize(1)),
			generator:    createFile,
			testData:     compressGzip(testData),
			wantErr:      true,
		},
		{
			name:         "tar gzip extraction",
			archiveName:  "test.tar.gz",
			expectedName: "test",
			cfg:          config.NewConfig(),
			generator:    createFile,
			testData: compressGzip(
				packTarWithContent([]tarContent{
					{Content: testData, Name: "test", Mode: 0640, Filetype: tar.TypeReg},
				})),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			// create testing directory
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.archiveName)

			// create a temporary file (if necessary)
			reader := tt.generator(tmpFile, tt.testData)
			defer func() {
				if closer, ok := reader.(io.Closer); ok {
					closer.Close()
				}
			}()

			// cancel context if necessary
			if tt.contextCanceled {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelCtx
			}

			// Unpack the file
			err := UnpackGZip(ctx, reader, tmpDir, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnpackGZip() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {

				// check if file was created
				if _, err := os.Stat(filepath.Join(tmpDir, tt.expectedName)); os.IsNotExist(err) {
					t.Errorf("UnpackGZip() file not created")
				}

				// check if file has the correct content
				data, err := os.ReadFile(filepath.Join(tmpDir, tt.expectedName))
				if err != nil {
					t.Errorf("UnpackGZip() error reading file: %v", err)
				}
				if !bytes.Equal(data, testData) {
					t.Errorf("%v: UnpackGZip() file content is not the expected", tt.name)
				}

			}
		})
	}
}

// compressGzip compresses data using gzip algorithm
func compressGzip(data []byte) []byte {
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	if _, err := gzWriter.Write(data); err != nil {
		panic(err)
	}
	if err := gzWriter.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
