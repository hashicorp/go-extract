package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestTarUnpack implements test cases
func TestTarUnpack(t *testing.T) {

	type TestfileGenerator func(string) string

	cases := []struct {
		name           string
		inputGenerator TestfileGenerator
		opts           []config.ConfigOption
		expectError    bool
	}{
		{
			name:           "unpack normal tar",
			inputGenerator: createTestTarNormal,
			opts:           []config.ConfigOption{},
			expectError:    false,
		},
		{
			name:           "unpack normal tar with 5 files",
			inputGenerator: createTestTarFiveFiles,
			opts:           []config.ConfigOption{},
			expectError:    false,
		},
		{
			name:           "unpack normal tar with 5 files, but file limit",
			inputGenerator: createTestTarFiveFiles,
			opts:           []config.ConfigOption{config.WithMaxFiles(4)},
			expectError:    true,
		},
		{
			name:           "unpack normal tar, but extraction time exceeded",
			inputGenerator: createTestTarNormal,
			opts:           []config.ConfigOption{config.WithMaxExtractionTime(0)},
			expectError:    true,
		},
		{
			name:           "unpack normal tar, but extraction size exceeded",
			inputGenerator: createTestTarNormal,
			opts:           []config.ConfigOption{config.WithMaxExtractionSize(1)},
			expectError:    true,
		},
		{
			name:           "unpack malicious tar, with traversal",
			inputGenerator: createTestTarWithPathTraversalInFile,
			opts:           []config.ConfigOption{},
			expectError:    true,
		},
		{
			name:           "unpack normal tar with symlink",
			inputGenerator: createTestTarWithSymlink,
			opts:           []config.ConfigOption{},
			expectError:    false,
		},
		{
			name:           "unpack normal tar with traversal symlink",
			inputGenerator: createTestTarWithPathTraversalSymlink,
			opts:           []config.ConfigOption{},
			expectError:    true,
		},
		{
			name:           "unpack normal tar with absolut path in symlink",
			inputGenerator: createTestTarWithAbsolutPathSymlink,
			opts:           []config.ConfigOption{},
			expectError:    true,
		},
		{
			name:           "malicous tar with symlink name path traversal",
			inputGenerator: createTestTarWithTraversalInSymlinkName,
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

			untarer := NewTar(config.NewConfig(tc.opts...))

			// perform actual tests
			input, _ := os.Open(tc.inputGenerator(testDir))
			want := tc.expectError
			err = untarer.Unpack(context.Background(), input, testDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}
}

// createTestTarNormal is a helper function to generate test content
func createTestTarNormal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarNormal.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare testfile for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, filepath.Base(f1.Name()), f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithSymlink is a helper function to generate test content
func createTestTarWithSymlink(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithSymlink.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
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

// createTestTarWithTraversalInSymlinkName is a helper function to generate test content
func createTestTarWithTraversalInSymlinkName(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithTraversalInSymlinkName.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
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

// createTestTarWithPathTraversalSymlink is a helper function to generate test content
func createTestTarWithPathTraversalSymlink(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithPathTraversalSymlink.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
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

// createTestTarWithAbsolutPathSymlink is a helper function to generate test content
func createTestTarWithAbsolutPathSymlink(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithAbsolutPathSymlink.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
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

// createTestTarWithPathTraversalInFile is a helper function to generate test content
func createTestTarWithPathTraversalInFile(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithPathTraversalInFile.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
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

// createTestTarFiveFiles is a helper function to generate test content
func createTestTarFiveFiles(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarFiveFiles.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	for i := 0; i < 5; i++ {

		// prepare testfile for be added to tar
		f1 := createTestFile(filepath.Join(tmpDir, fmt.Sprintf("test%d", i)), "foobar content")
		defer f1.Close()

		// add
		addFileToTarArchive(tarWriter, filepath.Base(f1.Name()), f1)
	}

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// addFileToTarArchive is a helper function to generate test content
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

// addLinkToTarArchive is a helper function to generate test content
func addLinkToTarArchive(tarWriter *tar.Writer, fileName string, linkTarget string) {
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

// createTar is a helper function to generate test content
func createTar(filePath string) *tar.Writer {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	return tar.NewWriter(f)
}
