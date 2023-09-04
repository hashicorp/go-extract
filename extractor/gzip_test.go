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

// TestZipUnpack test with various testcases the implementation of zip.Unpack
func TestGzipUnpack(t *testing.T) {

	type TestfileGenerator func(string) io.Reader

	cases := []struct {
		name           string
		inputGenerator TestfileGenerator
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
		{
			name:           "gzip with extraction time exceeded",
			inputGenerator: createTestGzipWithMoreContent,
			outputFileName: "test-gziped-file",
			opts:           []config.ConfigOption{config.WithMaxExtractionTime(0)},
			expectError:    true,
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

			gzipper := NewGzip(config.NewConfig(tc.opts...))

			// perform actual tests
			input := tc.inputGenerator(testDir)
			want := tc.expectError
			err = gzipper.Unpack(context.Background(), input, fmt.Sprintf("%s/%s", testDir, tc.outputFileName))
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

// createTestGzipWithFile creates a test gzip file in dstDir for testing
func createTestGzipWithFile(dstDir string) io.Reader {

	// define target
	targetFile := filepath.Join(dstDir, "GzipWithFile.gz")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare testfile for be added to zip
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
	gzWriter.Write([]byte(content))
	gzWriter.Close()

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

// TestGzipSuffix implements a test
func TestGzipSuffix(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		gzipper := NewGzip(config.NewConfig())
		want := ".gz"
		got := gzipper.FileSuffix()
		if got != want {
			t.Errorf("Unexpected filesuffix! want: %s, got :%s", want, got)
		}
	})
}

// TestGzipOffset implements a test
func TestGzipOffset(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		gzipper := NewGzip(config.NewConfig())
		want := 0
		got := gzipper.Offset()
		if got != want {
			t.Errorf("Unexpected offset! want: %d, got :%d", want, got)
		}
	})
}

// TestGzipSetConfig implements a test
func TestGzipSetConfig(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		gzipper := NewGzip(config.NewConfig())
		newConfig := config.NewConfig()
		gzipper.SetConfig(newConfig)

		// verify
		want := newConfig
		got := gzipper.config
		if got != want {
			t.Errorf("Config not adjusted!")
		}
	})
}

// TestGzipSetTarget implements a test
func TestGzipSetTarget(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		gzipper := NewGzip(config.NewConfig())
		newTarget := target.NewOs()
		gzipper.SetTarget(newTarget)

		// verify
		want := newTarget
		got := gzipper.target
		if got != want {
			t.Errorf("Target not adjusted!")
		}
	})
}

// TestGzipMagicBytes implements a test
func TestGzipMagicBytes(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		gzipper := NewGzip(config.NewConfig())
		want := [][]byte{
			{0x1f, 0x8b},
		}
		got := gzipper.magicBytes
		for idx := range got {
			for idy := range got[idx] {
				if got[idx][idy] != want[idx][idy] {
					t.Errorf("Magic byte missmatche!")
				}
			}
		}
	})
}
