package extractor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestZipUnpack test with various testcases the implementation of zip.Unpack
func TestGunzipUnpack(t *testing.T) {

	type TestfileGenerator func(string) string

	cases := []struct {
		name           string
		inputGenerator TestfileGenerator
		opts           []config.ConfigOption
		expectError    bool
	}{
		{
			name:           "normal gunzip with file",
			inputGenerator: createTestGunzipWithFile,
			opts:           []config.ConfigOption{config.WithOverwrite(true)},
			expectError:    false,
		},
		{
			name:           "gunzip with compressed txt",
			inputGenerator: createTestGunzipWithText,
			opts:           []config.ConfigOption{config.WithOverwrite(true)},
			expectError:    false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			gunzipper := NewGunzip(config.NewConfig(tc.opts...))

			// perform actual tests
			inputFile := tc.inputGenerator(testDir)
			outputFile := strings.TrimSuffix(filepath.Base(inputFile), ".gz")
			input, _ := os.Open(inputFile)
			want := tc.expectError
			err = gunzipper.Unpack(context.Background(), input, fmt.Sprintf("%s/%s", testDir, outputFile))
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}
}

// createGzip creates a gzip archive at dstFile with contents from input
func createGzip(dstFile string, input io.Reader) {
	// Create a new gzipped file
	gzippedFile, err := os.Create(dstFile)
	if err != nil {
		panic(err)
	}
	defer gzippedFile.Close()

	// Create a new gzip writer
	gzipWriter := gzip.NewWriter(gzippedFile)
	defer gzipWriter.Close()

	// Copy the contents of the original file to the gzip writer
	_, err = io.Copy(gzipWriter, input)
	if err != nil {
		panic(err)
	}

	// Flush the gzip writer to ensure all data is written
	gzipWriter.Flush()
}

// createTestGunzipWithFile creates a test gzip file in dstDir for testing
func createTestGunzipWithFile(dstDir string) string {

	// define target
	targetFile := filepath.Join(dstDir, "GunzipWithFile.gz")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare testfile for be added to zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// create Gzip file
	createGzip(targetFile, f1)

	// return path to zip
	return targetFile
}

// createTestGunzipWithText creates a test gzip file in dstDir for testing
func createTestGunzipWithText(dstDir string) string {

	// define target
	targetFile := filepath.Join(dstDir, "GunzipWithText.gz")

	// example text
	var bytesBuffer bytes.Buffer
	bytesBuffer.Write([]byte("some random content"))

	// create Gzip file
	createGzip(targetFile, &bytesBuffer)

	// return path to zip
	return targetFile
}
