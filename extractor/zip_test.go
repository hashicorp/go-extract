package extractor

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestZipUnpack test with various testcases the implementation of zip.Unpack
func TestZipUnpack(t *testing.T) {

	type TestfileGenerator func(string) string

	cases := []struct {
		name           string
		inputGenerator TestfileGenerator
		opts           []config.ConfigOption
		expectError    bool
	}{
		{
			name:           "normal zip",
			inputGenerator: createTestZipNormal,
			opts:           []config.ConfigOption{},
			expectError:    false,
		},
		{
			name:           "normal zip with 5 files",
			inputGenerator: createTestZipNormalFiveFiles,
			opts:           []config.ConfigOption{},
			expectError:    false,
		},
		{
			name:           "normal zip with 5 files, but extraction limit",
			inputGenerator: createTestZipNormalFiveFiles,
			opts:           []config.ConfigOption{config.WithMaxFiles(1)},
			expectError:    true,
		},
		{
			name:           "normal zip, but extraction time exceeded",
			inputGenerator: createTestZipNormalFiveFiles,
			opts:           []config.ConfigOption{config.WithMaxExtractionTime(0)},
			expectError:    true,
		},
		{
			name:           "normal zip, but limited extraction size of 1 byte",
			inputGenerator: createTestZipNormal,
			opts:           []config.ConfigOption{config.WithMaxExtractionSize(1)},
			expectError:    true,
		},
		{
			name:           "malicious zip with path traversal",
			inputGenerator: createTestZipPathtraversal,
			opts:           []config.ConfigOption{},
			expectError:    true,
		},
		{
			name:           "normal zip with symlink",
			inputGenerator: createTestZipWithSymlink,
			opts:           []config.ConfigOption{},
			expectError:    false,
		},
		{
			name:           "malicous zip with symlink target containing path traversal",
			inputGenerator: createTestZipWithSymlinkTargetPathTraversal,
			opts:           []config.ConfigOption{},
			expectError:    true,
		},
		{
			name:           "malicous zip with symlink target refering absolut path",
			inputGenerator: createTestZipWithSymlinkAbsolutPath,
			opts:           []config.ConfigOption{},
			expectError:    true,
		},
		{
			name:           "malicous zip with symlink name path traversal",
			inputGenerator: createTestZipWithSymlinkPathTraversalName,
			opts:           []config.ConfigOption{},
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

			unzipper := NewZip(config.NewConfig(tc.opts...))

			// perform actual tests
			input, _ := os.Open(tc.inputGenerator(testDir))
			want := tc.expectError
			err = unzipper.Unpack(context.Background(), input, testDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}
}

// createTestZipNormal creates a test zip file in dstDir for testing
func createTestZipNormal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipNormal.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
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

// createTestZipPathtraversal creates a test with a filename path traversal zip file in dstDir for testing
func createTestZipPathtraversal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipTraversal.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
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

// createTestZipNormalFiveFiles creates a test zip file with five files in dstDir for testing
func createTestZipNormalFiveFiles(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipNormalFiveFiles.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
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

// createTestZipWithSymlink creates a test zip file with a legit sym link in dstDir for testing
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

// createTestZipWithSymlinkPathTraversalName creates a test zip file, with a symlink, which filename contains a path traversal, in dstDir for testing
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

// createTestZipWithSymlinkAbsolutPath creates a test zip file, with a symlink to a absolut path, in dstDir for testing
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

// createTestZipWithSymlinkTargetPathTraversal creates a test zip file, with a path traversal in the link target, in dstDir for testing
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

// addLinkToZipArchive writes symlink linkName to linkTarget into zipWriter
func addLinkToZipArchive(zipWriter *zip.Writer, linkName string, linkTarget string) error {

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
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

// createZip creates a new zip file in filePath
func createZip(filePath string) *zip.Writer {
	targetFile := filepath.Join(filePath)
	archive, err := os.Create(targetFile)
	if err != nil {
		panic(err)
	}
	return zip.NewWriter(archive)
}

// createTestFile creates a file under path containing content
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

// createTestFIFO creates a fifo under path
func createTestFIFO(path string) {
	// create only on non-windows systems
	if runtime.GOOS != "windows" {
		if err := syscall.Mkfifo(path, 0640); err != nil {
			panic(err)
		}
	}
}
