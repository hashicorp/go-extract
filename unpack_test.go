package extract

import (
	"archive/tar"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/extractor"
)

func TestFindExtractor(t *testing.T) {
	// test cases
	cases := []struct {
		name     string
		input    string
		expected Extractor
	}{
		{
			name:     "get zip extractor from file",
			input:    "foo.zip",
			expected: extractor.NewZip(nil),
		},
		{
			name:     "get zip extractor from file in path",
			input:    "foo.zip",
			expected: extractor.NewZip(nil),
		},
		{
			name:     "get tar extractor from file",
			input:    "foo.tar",
			expected: extractor.NewTar(nil),
		},
		{
			name:     "get tar extractor from file in path",
			input:    "foo.tar",
			expected: extractor.NewTar(nil),
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
			expected: extractor.NewZip(nil),
		},
		{
			name:     "camel case",
			input:    "foo.TaR",
			expected: extractor.NewTar(nil),
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			// prepare vars
			var failed bool
			want := tc.expected

			// perform actual tests
			got := findExtractor(tc.input)

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
		opts           []ExtractorOption
		expectError    bool
	}{
		{
			name:           "normal zip",
			inputGenerator: createTestZipNormal,
			opts:           []ExtractorOption{},
			expectError:    false,
		},
		{
			name:           "normal zip with 5 files",
			inputGenerator: createTestZipNormalFiveFiles,
			opts:           []ExtractorOption{},
			expectError:    false,
		},
		{
			name:           "normal zip with 5 files, but extraction limit",
			inputGenerator: createTestZipNormalFiveFiles,
			opts:           []ExtractorOption{WithMaxFiles(1)},
			expectError:    true,
		},
		{
			name:           "normal zip, but extraction time exceeded",
			inputGenerator: createTestZipNormal,
			opts:           []ExtractorOption{WithMaxExtractionTime(0)},
			expectError:    true,
		},
		{
			name:           "normal zip, but limited extraction size of 1 byte",
			inputGenerator: createTestZipNormal,
			opts:           []ExtractorOption{WithMaxFileSize(1)},
			expectError:    true,
		},
		{
			name:           "malicious zip with path traversal",
			inputGenerator: createTestZipPathtraversal,
			opts:           []ExtractorOption{},
			expectError:    true,
		},
		{
			name:           "normal zip with symlink",
			inputGenerator: createTestZipWithSymlink,
			opts:           []ExtractorOption{},
			expectError:    false,
		},
		{
			name:           "malicous zip with symlink target containing path traversal",
			inputGenerator: createTestZipWithSymlinkTargetPathTraversal,
			opts:           []ExtractorOption{},
			expectError:    true,
		},
		{
			name:           "malicous zip with symlink target refering absolut path",
			inputGenerator: createTestZipWithSymlinkAbsolutPath,
			opts:           []ExtractorOption{},
			expectError:    true,
		},
		{
			name:           "malicous zip with symlink name path traversal",
			inputGenerator: createTestZipWithSymlinkPathTraversalName,
			opts:           []ExtractorOption{},
			expectError:    true,
		},
		{
			name:           "unpack normal tar",
			inputGenerator: createTestTarNormal,
			opts:           []ExtractorOption{},
			expectError:    false,
		},
		{
			name:           "unpack normal tar with 5 files",
			inputGenerator: createTestTarFiveFiles,
			opts:           []ExtractorOption{},
			expectError:    false,
		},
		{
			name:           "unpack normal tar with 5 files, but file limit",
			inputGenerator: createTestTarFiveFiles,
			opts:           []ExtractorOption{WithMaxFiles(4)},
			expectError:    true,
		},
		{
			name:           "unpack normal tar, but extraction time exceeded",
			inputGenerator: createTestTarNormal,
			opts:           []ExtractorOption{WithMaxExtractionTime(0)},
			expectError:    true,
		},
		{
			name:           "unpack normal tar, but extraction size exceeded",
			inputGenerator: createTestTarNormal,
			opts:           []ExtractorOption{WithMaxFileSize(1)},
			expectError:    true,
		},
		{
			name:           "unpack malicious tar, with traversal",
			inputGenerator: createTestTarWithPathTraversalInFile,
			opts:           []ExtractorOption{},
			expectError:    true,
		},
		{
			name:           "unpack normal tar with symlink",
			inputGenerator: createTestTarWithSymlink,
			opts:           []ExtractorOption{},
			expectError:    false,
		},
		{
			name:           "unpack normal tar with traversal symlink",
			inputGenerator: createTestTarWithPathTraversalSymlink,
			opts:           []ExtractorOption{},
			expectError:    true,
		},
		{
			name:           "unpack normal tar with absolut path in symlink",
			inputGenerator: createTestTarWithAbsolutPathSymlink,
			opts:           []ExtractorOption{},
			expectError:    true,
		},
		{
			name:           "malicous tar with symlink name path traversal",
			inputGenerator: createTestTarWithTraversalInSymlinkName,
			opts:           []ExtractorOption{},
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
			err = Unpack(context.Background(), input, testDir, tc.opts...)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}
}

func createTestTarNormal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarNormal.tar")

	// create a temporary dir for files in tar archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

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

func createTestTarWithSymlink(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithSymlink.tar")

	// create a temporary dir for files in tar archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(tarWriter, "testLink", "testTarget")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}
func createTestTarWithTraversalInSymlinkName(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithTraversalInSymlinkName.tar")

	// create a temporary dir for files in tar archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(tarWriter, "../testLink", "testTarget")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

func createTestTarWithPathTraversalSymlink(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithPathTraversalSymlink.tar")

	// create a temporary dir for files in tar archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(tarWriter, "testLink", "../testTarget")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}
func createTestTarWithAbsolutPathSymlink(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithAbsolutPathSymlink.tar")

	// create a temporary dir for files in tar archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(tarWriter, "testLink", "/tmp/test")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

func createTestTarWithPathTraversalInFile(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithPathTraversalInFile.tar")

	// create a temporary dir for files in tar archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare testfile for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// add
	addFileToTarArchive(tarWriter, "../test", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

func createTestTarFiveFiles(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarFiveFiles.tar")

	// create a temporary dir for files in tar archive
	tmpDir := createTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	for i := 0; i < 5; i++ {

		// prepare testfile for be added to tar
		f1 := createTestFile(filepath.Join(tmpDir, fmt.Sprintf("test%d", i)), "foobar content")
		defer f1.Close()

		// add
		addFileToTarArchive(tarWriter, f1.Name(), f1)
	}

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

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
	if err := addLinkToZipArchive(zipWriter, "legitLinkName", "legitLinkTarget"); err != nil {
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
	if err := addLinkToZipArchive(zipWriter, "../malicousLink", "nirvana"); err != nil {
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
	if err := addLinkToZipArchive(zipWriter, "maliciousLink", "/etc/passwd"); err != nil {
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
	if err := addLinkToZipArchive(zipWriter, "maliciousLink", "../malicousLinkTarget"); err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

func addLinkToZipArchive(zipWriter *zip.Writer, linkName string, linkTarget string) error {

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

func addLinkToTarArchive(tarWriter *tar.Writer, fileName string, linkTarget string) {
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

	// create a new dir/file header
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		panic(err)
	}

	// adjust file headers
	header.Name = fileName
	header.Linkname = linkTarget

	if err := tarWriter.WriteHeader(header); err != nil {
		panic(err)
	}
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
