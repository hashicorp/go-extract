package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
	"github.com/hashicorp/go-extract/target"
)

// TestFindExtractor implements test cases
func TestFindExtractor(t *testing.T) {
	// test cases
	cases := []struct {
		name           string
		createTestFile func(string) string
		expected       Extractor
	}{
		{
			name:           "get zip extractor from file",
			createTestFile: createTestZip,
			expected:       extractor.NewZip(),
		},
		{
			name:           "get tar extractor from file",
			createTestFile: createTestTar,
			expected:       extractor.NewTar(),
		},
		{
			name:           "get gzip extractor from file",
			createTestFile: createTestGzipWithFile,
			expected:       extractor.NewGzip(),
		},
		{
			name:           "get nil extractor fot textfile",
			createTestFile: createTestNonArchive,
			expected:       nil,
		},
	}

	// create testing directory
	testDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		t.Errorf(err.Error())
	}
	testDir = filepath.Clean(testDir) + string(os.PathSeparator)
	defer os.RemoveAll(testDir)

	// run cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// prepare vars
			want := tc.expected

			// perform actual tests
			f, err := os.Open(tc.createTestFile(testDir))
			if err != nil {
				t.Fatal(err)
			}
			input, err := io.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			got := findExtractor(input)

			// success if both are nil and no engine found
			if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", want) {
				t.Fatalf("expected: %v\ngot: %v", want, got)
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
func createTestGzipWithFile(dstDir string) string {

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

	// return path to zip
	return targetFile
}

// createTestZip is a helper function to generate test data
func createTestZip(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TestZip.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	archive, _ := os.Create(targetFile)
	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	// prepare testfile for be added to zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// write file into zip
	w1, _ := zipWriter.Create("test")
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	// return path to zip
	return targetFile
}

// createTestNonArchive is a helper function to generate test data
func createTestNonArchive(dstDir string) string {
	targetFile := filepath.Join(dstDir, "test.txt")
	createTestFile(targetFile, "foo bar test")
	return targetFile
}

// createTestFile is a helper function to generate test files
func createTestFile(path string, content string) *os.File {
	byteArray := []byte(content)
	err := os.WriteFile(path, byteArray, 0644)
	if err != nil {
		panic(err)
	}
	newFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	return newFile
}

// createTestTar is a helper function to generate test data
func createTestTar(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarNormal.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer

	f, _ := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	tarWriter := tar.NewWriter(f)

	// prepare testfile for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, filepath.Base(f1.Name()), f1)

	// close tar
	tarWriter.Close()

	// return path to tar
	return targetFile
}

// addFileToTarArchive is a helper function
func addFileToTarArchive(tarWriter *tar.Writer, fileName string, f1 *os.File) {

	fileInfo, err := os.Lstat(f1.Name())
	if err != nil {
		panic(err)
	}

	// create a new dir/file header
	header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
	if err != nil {
		panic(err)
	}

	// adjust filename
	header.Name = fileName

	// write the header
	if err := tarWriter.WriteHeader(header); err != nil {
		panic(err)
	}

	// add content
	if _, err := io.Copy(tarWriter, f1); err != nil {
		panic(err)
	}
}

// TestUnpack is a test function
func TestUnpack(t *testing.T) {
	cases := []struct {
		name        string
		fn          func(string) string
		expectError bool
	}{
		{
			name:        "get zip extractor from file",
			fn:          createTestZip,
			expectError: false,
		},
		{
			name:        "get tar extractor from file",
			fn:          createTestTar,
			expectError: false,
		},
		{
			name:        "get gzip extractor from file",
			fn:          createTestGzipWithFile,
			expectError: false,
		},
		{
			name:        "get nil extractor fot textfile",
			fn:          createTestNonArchive,
			expectError: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "fooo*")
			if err != nil {
				panic(err)
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			// prepare vars
			want := tc.expectError

			// perform actual tests
			archive, err := os.Open(tc.fn(testDir))
			if err != nil {
				panic(err)
			}
			err = Unpack(
				context.Background(),
				archive,
				testDir,
				target.NewOs(),
				config.NewConfig(
					config.WithOverwrite(true),
				),
			)
			got := err != nil

			// success if both are nil and no engine found
			if want != got {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, want, err)
			}
		})
	}
}

func TestMatchesMagicBytes(t *testing.T) {
	cases := []struct {
		name        string
		data        []byte
		magicBytes  []byte
		offset      int
		expectMatch bool
	}{
		{
			name:        "match",
			data:        []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			magicBytes:  []byte{0x02, 0x03},
			offset:      2,
			expectMatch: true,
		},
		{
			name:        "missmatch",
			data:        []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			magicBytes:  []byte{0x02, 0x03},
			offset:      1,
			expectMatch: false,
		},
		{
			name:        "to few data to match",
			data:        []byte{0x00},
			magicBytes:  []byte{0x02, 0x03},
			offset:      1,
			expectMatch: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			expected := tc.expectMatch
			got := matchesMagicBytes(tc.data, tc.offset, tc.magicBytes)

			// success if both are nil and no engine found
			if got != expected {
				t.Errorf("test case %d failed: %s!", i, tc.name)
			}
		})
	}
}
