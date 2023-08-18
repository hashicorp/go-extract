package extractor

import (
	"archive/tar"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract"
)

func TestFindExtractor(t *testing.T) {
	// test cases
	cases := []struct {
		name     string
		input    string
		expected extract.Extractor
	}{
		{
			name:     "get zip extractor from file",
			input:    "foo.zip",
			expected: NewZip(extract.Default()),
		},
		{
			name:     "get zip extractor from file in path",
			input:    "foo.zip",
			expected: NewZip(extract.Default()),
		},
		{
			name:     "get tar extractor from file",
			input:    "foo.tar",
			expected: NewTar(extract.Default()),
		},
		{
			name:     "get tar extractor from file in path",
			input:    "foo.tar",
			expected: NewTar(extract.Default()),
		},
		{
			name:     "unspported file type .7z",
			input:    "foo.7z",
			expected: nil,
		},
		{
			name:     "no filetype",
			input:    "foo",
			expected: nil,
		},
		{
			name:     "camel case",
			input:    "foo.zIp",
			expected: NewZip(extract.Default()),
		},
		{
			name:     "camel case",
			input:    "foo.TaR",
			expected: NewTar(extract.Default()),
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			// prepare vars
			var failed bool
			want := tc.expected

			// perform actual tests
			got := findExtractor(extract.Default(), tc.input)

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

func TestUnpack(t *testing.T) {

	type TestfileGenerator func(string) string

	cases := []struct {
		name           string
		inputGenerator TestfileGenerator
		config         *extract.Config
		expectError    bool
	}{
		{
			name:           "normal zip",
			inputGenerator: createTestZipNormal,
			config:         extract.Default(),
			expectError:    false,
		},
		{
			name:           "normal zip with 5 files",
			inputGenerator: createTestZipNormalFiveFiles,
			config:         extract.Default(),
			expectError:    false,
		},
		{
			name:           "normal zip with 5 files, but extraction limit",
			inputGenerator: createTestZipNormalFiveFiles,
			config:         &extract.Config{MaxFiles: 1, MaxExtractionTime: -1, MaxFileSize: -1},
			expectError:    true,
		},
		{
			name:           "normal zip, but extraction time exceeded",
			inputGenerator: createTestZipNormal,
			config:         &extract.Config{MaxFiles: -1, MaxExtractionTime: 0, MaxFileSize: -1},
			expectError:    true,
		},
		{
			name:           "normal zip, but limited extraction size of 1 byte",
			inputGenerator: createTestZipNormal,
			config:         &extract.Config{MaxFiles: -1, MaxExtractionTime: -1, MaxFileSize: 1},
			expectError:    true,
		},
		{
			name:           "malicious zip with path traversal",
			inputGenerator: createTestZipPathtraversal,
			config:         extract.Default(),
			expectError:    true,
		},
		{
			name:           "normal zip with symlink",
			inputGenerator: createTestZipWithSymlink,
			config:         extract.Default(),
			expectError:    false,
		},
		{
			name:           "malicous zip with symlink target containing path traversal",
			inputGenerator: createTestZipWithSymlinkTargetPathTraversal,
			config:         extract.Default(),
			expectError:    true,
		},
		{
			name:           "malicous zip with symlink target refering absolut path",
			inputGenerator: createTestZipWithSymlinkAbsolutPath,
			config:         extract.Default(),
			expectError:    true,
		},
		{
			name:           "malicous zip with symlink name path traversal",
			inputGenerator: createTestZipWithSymlinkPathTraversalName,
			config:         extract.Default(),
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

			// perform actual tests
			input := tc.inputGenerator(testDir)
			want := tc.expectError
			err = UnpackWithConfig(context.Background(), tc.config, input, testDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}
}

// func createTestTarNormal(dstDir string) string {

// 	targetFile := filepath.Join(dstDir, "TarNormal.tar")

// 	// create a temporary dir for files in zip archive
// 	tmpDir := createTmpDir()
// 	defer os.RemoveAll(tmpDir)

// 	// prepare generated zip+writer
// 	tarWriter := createTar(targetFile)

// 	// prepare testfile for be added to tar
// 	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
// 	defer f1.Close()

// 	// write file into tar

// 	// create a new dir/file header
// 	header, err := tar.FileInfoHeader(f1., fi.Name())
// 	if err != nil {
// 		return err
// 	}

// 	// update the name to correctly reflect the desired destination when untaring
// 	header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

// 	// write the header
// 	if err := tw.WriteHeader(header); err != nil {
// 		return err
// 	}

// 	w1, err := tarWriter.Create("test")
// 	if err != nil {
// 		panic(err)
// 	}
// 	if _, err := io.Copy(w1, f1); err != nil {
// 		panic(err)
// 	}

// 	// close zip
// 	tarWriter.Close()

// 	// return path to zip
// 	return targetFile
// }

func createTestZipNormal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipNormal.zip")

	// create a temporary dir for files in zip archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// prepare testfile for be added to zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// write file into zip
	w1, err := zipWriter.Create("test")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func createTestZipPathtraversal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipTraversal.zip")

	// create a temporary dir for files in zip archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// prepare testfile for be added to zipzip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// write file into zip
	w1, err := zipWriter.Create("../test")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func createTestZipNormalFiveFiles(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipNormalFiveFiles.zip")

	// create a temporary dir for files in zip archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	for i := 0; i < 5; i++ {
		// prepare testfile for be added to zip
		fName := fmt.Sprintf("test%d", i)
		f1 := createTestFile(filepath.Join(tmpDir, fName), "foobar content")
		defer f1.Close()

		// write file into zip
		w1, err := zipWriter.Create(fName)
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(w1, f1); err != nil {
			panic(err)
		}
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func createTestZipWithSymlink(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipNormalWithSymlink.zip")

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// add link to archive
	if err := addLinkToArchive(zipWriter, "legitLinkName", "legitLinkTarget"); err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func createTestZipWithSymlinkPathTraversalName(dstDir string) string {

	targetFile := filepath.Join(dstDir, "createTestZipWithSymlinkPathTraversalName.zip")

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// add link to archive
	if err := addLinkToArchive(zipWriter, "../malicousLink", "nirvana"); err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func createTestZipWithSymlinkAbsolutPath(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipWithSymlinkTargetAbsolutPath.zip")

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// add link to archive
	if err := addLinkToArchive(zipWriter, "maliciousLink", "/etc/passwd"); err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func createTestZipWithSymlinkTargetPathTraversal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipWithSymlinkTargetPathTraversal.zip")

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// add link to archive
	if err := addLinkToArchive(zipWriter, "maliciousLink", "../malicousLinkTarget"); err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func addLinkToArchive(zipWriter *zip.Writer, linkName string, linkTarget string) error {

	// create a temporary dir for files in zip archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// create dummy link to get data structure
	dummyLink := filepath.Join(tmpDir, "dummylink")
	if err := os.Symlink("nirvana", dummyLink); err != nil {
		panic(err)
	}

	// get file stats for testing operating system
	info, err := os.Lstat(dummyLink)
	if err != nil {
		panic(err)
	}

	// get file header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		panic(err)
	}

	// adjust file headers
	header.Name = linkName
	header.Method = zip.Deflate

	// create writer for link
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		panic(err)
	}

	// Write symlink's target to writer - file's body for symlinks is the symlink target.
	_, err = writer.Write([]byte(linkTarget))
	if err != nil {
		return err
	}

	return nil
}

func createTar(filePath string) *tar.Writer {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	return tar.NewWriter(f)
}

func createZip(filePath string) *zip.Writer {
	targetFile := filepath.Join(filePath)
	archive, err := os.Create(targetFile)
	if err != nil {
		panic(err)
	}
	return zip.NewWriter(archive)
}

func createTmpDir() string {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		panic(err)
	}
	return tmpDir
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
