package extractor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestZipUnpack test with various test cases the implementation of zip.Unpack
func TestGzipUnpack(t *testing.T) {

	type TestFileGenerator func(string) io.Reader

	cases := []struct {
		name           string
		inputGenerator TestFileGenerator
		outputFileName string
		opts           []config.ConfigOption
		expectError    bool
	}{
		{
			name:           "normal gzip with file",
			inputGenerator: createTestGzipWithFile,
			outputFileName: "test-gziped-file",
			opts:           []config.ConfigOption{config.WithOverwrite(true)},
			expectError:    false,
		},
		{
			name:           "gzip with compressed txt",
			inputGenerator: createTestGzipWithText,
			outputFileName: "",
			opts:           []config.ConfigOption{config.WithOverwrite(true)},
			expectError:    false,
		},
		{
			name:           "gzip with limited extraction size",
			inputGenerator: createTestGzipWithMoreContent,
			outputFileName: "test-gziped-file",
			opts:           []config.ConfigOption{config.WithMaxExtractionSize(512)},
			expectError:    true,
		},
		{
			name:           "gzip with unlimited extraction size",
			inputGenerator: createTestGzipWithMoreContent,
			outputFileName: "test-gziped-file",
			opts:           []config.ConfigOption{config.WithMaxExtractionSize(-1)},
			expectError:    false,
		},
		// TODO: use context for timeout
		// {
		// 	name:           "gzip with extraction time exceeded",
		// 	inputGenerator: createTestGzipWithMoreContent,
		// 	outputFileName: "test-gziped-file",
		// 	opts:           []config.ConfigOption{config.WithMaxExtractionTime(0)},
		// 	expectError:    true,
		// },
		{
			name:           "tar gzip",
			inputGenerator: createTestTarGzipWithFile,
			outputFileName: "",
			opts:           []config.ConfigOption{},
			expectError:    false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			gziper := NewGzip()

			// perform actual tests
			input := tc.inputGenerator(testDir)
			want := tc.expectError
			err = gziper.Unpack(context.Background(), input, fmt.Sprintf("%s%s", testDir, tc.outputFileName), target.NewOs(), config.NewConfig(tc.opts...))
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}
}

// createGzip creates a gzip archive at dstFile with contents from input
func createGzip(dstFile string, input io.Reader) {
	// Create a new gziped file
	gzipedFile, err := os.Create(dstFile)
	if err != nil {
		panic(err)
	}
	defer gzipedFile.Close()

	// Create a new gzip writer
	gzipWriter := gzip.NewWriter(gzipedFile)
	defer gzipWriter.Close()

	// Copy the contents of the original file to the gzip writer
	_, err = io.Copy(gzipWriter, input)
	if err != nil {
		panic(err)
	}

	// Flush the gzip writer to ensure all data is written
	gzipWriter.Flush()
}

// createTestGzipWithFile creates a test gzip file in dstDir for testing
func createTestGzipWithFile(dstDir string) io.Reader {

	// define target
	targetFile := filepath.Join(dstDir, "GzipWithFile.gz")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare test file for be added to zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// create Gzip file
	createGzip(targetFile, f1)

	// return reader
	file, err := os.Open(targetFile)
	if err != nil {
		panic(err)
	}
	return file
}

// createTestGzipWithText creates a test gzip file in dstDir for testing
func createTestGzipWithText(dstDir string) io.Reader {

	content := "some random content"
	// Initialize gzip
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	if _, err := gzWriter.Write([]byte(content)); err != nil {
		panic(err)
	}
	if err := gzWriter.Close(); err != nil {
		panic(err)
	}

	return buf
}

func RandStringBytes(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// createTestGzipWithMoreContent creates a test gzip file in dstDir for testing
func createTestGzipWithMoreContent(dstDir string) io.Reader {

	// define target
	targetFile := filepath.Join(dstDir, "GzipWithMoreContent.gz")

	// example text
	var bytesBuffer bytes.Buffer
	bytesBuffer.Write([]byte(RandStringBytes(1 << (10 * 2)))) // Generate 1 Mb text

	// create Gzip file
	createGzip(targetFile, &bytesBuffer)

	// return reader
	file, err := os.Open(targetFile)
	if err != nil {
		panic(err)
	}
	return file
}

// createTestGzipWithFile creates a test gzip file in dstDir for testing
func createTestTarGzipWithFile(dstDir string) io.Reader {

	// define target
	targetFile := filepath.Join(dstDir, "GzipWithTarGz.tar.gz")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// get test tar
	tarFile := createTestTarNormal(tmpDir)

	tarReader, err := os.Open(tarFile)
	if err != nil {
		panic(err)
	}
	defer tarReader.Close()

	// create Gzip file
	createGzip(targetFile, tarReader)

	// return reader
	file, err := os.Open(targetFile)
	if err != nil {
		panic(err)
	}
	return file
}
