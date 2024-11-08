package extract_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
)

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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := extract.IsGZip(test.header); got != test.want {
				t.Errorf("IsGZIP() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGzipUnpack(t *testing.T) {
	testData := []byte("Hello, World!")

	tests := []struct {
		name            string
		archiveName     string
		expectedName    string
		cfg             *extract.Config
		generator       func(target string, data []byte) io.Reader
		testData        []byte
		contextCanceled bool
		wantErr         bool
	}{
		{
			name:         "normal gzip with file",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			testData:     compressGzip(t, testData),
			wantErr:      false,
		},
		{
			name:         "random file with no gzip",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			testData:     testData,
			wantErr:      true,
		},
		{
			name:         "gzip error while reading the header",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			testData:     []byte("123"),
			wantErr:      true,
		},
		{
			name:            "gzip with canceled context",
			archiveName:     "test.gz",
			expectedName:    "test",
			cfg:             extract.NewConfig(),
			generator:       newTestFile,
			testData:        compressGzip(t, testData),
			contextCanceled: true,
			wantErr:         true,
		},
		{
			name:         "gzip with limited reader",
			archiveName:  "test.gz",
			expectedName: "test",
			cfg:          extract.NewConfig(extract.WithMaxInputSize(1)),
			generator:    newTestFile,
			testData:     compressGzip(t, testData),
			wantErr:      true,
		},
		{
			name:         "tar gzip extraction",
			archiveName:  "test.tar.gz",
			expectedName: "test",
			cfg:          extract.NewConfig(),
			generator:    newTestFile,
			testData: compressGzip(t, packTar(t, []archiveContent{
				{Content: testData, Name: "test", Mode: 0640, Filetype: tar.TypeReg},
			}),
			),
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			// Create a new target
			testingTarget := extract.NewDisk()

			// create testing directory
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, test.archiveName)

			// create a temporary file (if necessary)
			reader := test.generator(tmpFile, test.testData)
			defer func() {
				if closer, ok := reader.(io.Closer); ok {
					closer.Close()
				}
			}()

			// cancel context if necessary
			if test.contextCanceled {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelCtx
			}

			// Unpack the file
			err := extract.UnpackGZip(ctx, testingTarget, tmpDir, reader, test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("UnpackGZip() error = %v, wantErr %v", err, test.wantErr)
			}

			if !test.wantErr {

				// check if file was created
				if _, err := os.Stat(filepath.Join(tmpDir, test.expectedName)); os.IsNotExist(err) {
					t.Errorf("UnpackGZip() file not created")
				}

				// check if file has the correct content
				data, err := os.ReadFile(filepath.Join(tmpDir, test.expectedName))
				if err != nil {
					t.Errorf("UnpackGZip() error reading file: %v", err)
				}
				if !bytes.Equal(data, testData) {
					t.Errorf("%v: UnpackGZip() file content is not the expected", test.name)
				}

			}
		})
	}
}
