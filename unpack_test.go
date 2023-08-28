package extract

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
	"github.com/hashicorp/go-extract/target"
)

func TestFindExtractor(t *testing.T) {

	type TestfileGenerator func(string) string

	// test cases
	cases := []struct {
		name     string
		fkt      TestfileGenerator
		expected Extractor
	}{
		{
			name:     "get zip extractor from file",
			fkt:      createTestZip,
			expected: extractor.NewZip(config.NewConfig()),
		},
		{
			name:     "get tar extractor from file",
			fkt:      createTestTar,
			expected: extractor.NewTar(config.NewConfig()),
		},
		{
			name:     "get nil extractor fot textfile",
			fkt:      createTestNonArchive,
			expected: nil,
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

			// prepare vars
			var failed bool
			want := tc.expected

			// perform actual tests
			f, err := os.Open(tc.fkt(testDir))
			input, err := io.ReadAll(f)

			if err != nil {
				panic(err)
			}
			got := findExtractor(input)

			// success if both are nil and no engine found
			if want == got {
				return
			}

			// check if engine detection failed
			if got == nil {
				failed = true
			}

			// if not failed yet, compare identified suffixes
			if !failed {
				if got.FileSuffix() != want.FileSuffix() {
					failed = true
				}
			}

			if failed {
				t.Errorf("test case %d failed: %s\nexpected: %v\ngot: %v", i, tc.name, want, got)
			}

		})
	}

}

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
	io.Copy(w1, f1)

	// return path to zip
	return targetFile
}

func createTestNonArchive(dstDir string) string {
	targetFile := filepath.Join(dstDir, "test.txt")
	createTestFile(targetFile, "foo bar test")
	return targetFile
}

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
	addFileToTarArchive(tarWriter, f1.Name(), f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

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
