package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestTarUnpack implements test cases
func TestTarUnpack(t *testing.T) {

	// generate cancled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	cases := []struct {
		name              string
		testFileGenerator func(*testing.T, string) string
		opts              []config.ConfigOption
		expectError       bool
		ctx               context.Context
	}{
		{
			name:              "unpack normal tar",
			testFileGenerator: createTestTarNormal,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "unpack normal tar, but pattern mismatch",
			testFileGenerator: createTestTarNormal,
			opts:              []config.ConfigOption{config.WithPatterns("*foo")},
			expectError:       false,
		},
		{
			name:              "unpack normal tar, but context timeout",
			testFileGenerator: createTestTarNormal,
			opts:              []config.ConfigOption{},
			ctx:               canceledCtx,
			expectError:       true,
		},
		{
			name:              "unpack normal tar with 5 files",
			testFileGenerator: createTestTarFiveFiles,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "unpack normal tar with 5 files, but file limit",
			testFileGenerator: createTestTarFiveFiles,
			opts:              []config.ConfigOption{config.WithMaxFiles(4)},
			expectError:       true,
		},
		// TODO: use context for timeout
		// {
		// 	name:           "unpack normal tar, but extraction time exceeded",
		// 	inputGenerator: createTestTarNormal,
		// 	opts:           []config.ConfigOption{config.WithMaxExtractionTime(0)},
		// 	expectError:    true,
		// },
		{
			name:              "unpack normal tar, but extraction size exceeded",
			testFileGenerator: createTestTarNormal,
			opts:              []config.ConfigOption{config.WithMaxExtractionSize(1)},
			expectError:       true,
		},
		{
			name:              "unpack malicious tar, with traversal",
			testFileGenerator: createTestTarWithPathTraversalInFile,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "unpack normal tar with symlink",
			testFileGenerator: createTestTarWithSymlink,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "unpack tar with traversal in directory",
			testFileGenerator: createTestTarWithTraversalInDirectory,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "unpack tar with traversal in directory",
			testFileGenerator: createTestTarWithTraversalInDirectory,
			opts:              []config.ConfigOption{config.WithContinueOnError(true)},
			expectError:       false,
		},
		{
			name:              "unpack normal tar with traversal symlink",
			testFileGenerator: createTestTarWithPathTraversalSymlink,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "unpack normal tar with symlink, but symlinks are denied",
			testFileGenerator: createTestTarWithSymlink,
			opts:              []config.ConfigOption{config.WithAllowSymlinks(false)},
			expectError:       true,
		},
		{
			name:              "unpack normal tar with symlink, but symlinks are denied, but continue on error",
			testFileGenerator: createTestTarWithSymlink,
			opts:              []config.ConfigOption{config.WithAllowSymlinks(false), config.WithContinueOnError(true)},
			expectError:       false,
		},
		{
			name:              "unpack normal tar with symlink, but symlinks are denied, but continue on unsupported files",
			testFileGenerator: createTestTarWithSymlink,
			opts:              []config.ConfigOption{config.WithAllowSymlinks(false), config.WithContinueOnUnsupportedFiles(true)},
			expectError:       false,
		},

		{
			name:              "unpack normal tar with absolute path in symlink",
			testFileGenerator: createTestTarWithAbsolutePathSymlink,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "malicious tar with symlink name path traversal",
			testFileGenerator: createTestTarWithTraversalInSymlinkName,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "malicious tar.gz with empty name for compressed file",
			testFileGenerator: createTestTarGzWithEmptyFileName,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "malicious tar with .. as filename",
			testFileGenerator: createTestTarDotDotFilename,
			opts:              []config.ConfigOption{config.WithOverwrite(true)},
			expectError:       true,
		},
		{
			name:              "malicious tar with FIFO filetype",
			testFileGenerator: createTestTarWithFIFO,
			opts:              []config.ConfigOption{config.WithOverwrite(true)},
			expectError:       true,
		},
		{
			name:              "malicious tar with zip slip attack",
			testFileGenerator: createTestTarWithZipSlip,
			opts:              []config.ConfigOption{config.WithContinueOnError(false)},
			expectError:       true,
		}, {
			name:              "absolute path in filename",
			testFileGenerator: createTestTarWithAbsolutePathInFilename,
			opts:              []config.ConfigOption{},
			expectError:       false,
		}, {
			name:              "absolute path in filename (windows)",
			testFileGenerator: createTestTarWithAbsolutePathInFilenameWindows,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			testDir := t.TempDir()

			untarer := NewTar()
			if tc.ctx == nil {
				tc.ctx = context.Background()
			}
			ctx := tc.ctx

			// perform actual tests
			input, _ := os.Open(tc.testFileGenerator(t, testDir))
			want := tc.expectError
			err := untarer.Unpack(ctx, input, testDir, target.NewOS(), config.NewConfig(tc.opts...))
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}
}

// createTestTarNormal is a helper function to generate test content
func createTestTarNormal(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarNormal.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare test file for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, "test", f1)

	// add empty dir to tar
	addFileToTarArchive(tarWriter, "emptyDir/", nil)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

func createTestTarWithTraversalInDirectory(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "TarWithTraversalInDirectory.tar")

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add dir with traversal
	addFileToTarArchive(tarWriter, "../test", nil)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

func createTestTarWithFIFO(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithFIFO.tar")

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// create fifo and add
	path := path.Join(dstDir, "testFIFO")
	createTestFile(path, "ignored anyway")
	addFifoToTarArchive(tarWriter, "fifo", path)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarDotDotFilename is a helper function to generate test content
func createTestTarDotDotFilename(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarDotDot.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare test file for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, "..", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithSymlink is a helper function to generate test content
func createTestTarWithSymlink(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithSymlink.tar")

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(t, tarWriter, "testLink", "testTarget")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithSymlink is a helper function to generate test content
func createTestTarWithZipSlip(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithZipSlip.tar")

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlinks
	addLinkToTarArchive(t, tarWriter, "sub/to-parent", "../")
	addLinkToTarArchive(t, tarWriter, "sub/to-parent/one-above", "../")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithTraversalInSymlinkName is a helper function to generate test content
func createTestTarWithTraversalInSymlinkName(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithTraversalInSymlinkName.tar")

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(t, tarWriter, "../testLink", "testTarget")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithPathTraversalSymlink is a helper function to generate test content
func createTestTarWithPathTraversalSymlink(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithPathTraversalSymlink.tar")

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(t, tarWriter, "testLink", "../testTarget")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithAbsolutePathSymlink is a helper function to generate test content
func createTestTarWithAbsolutePathSymlink(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithAbsolutePathSymlink.tar")

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlink
	addLinkToTarArchive(t, tarWriter, "testLink", "/tmp/test")

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithPathTraversalInFile is a helper function to generate test content
func createTestTarWithPathTraversalInFile(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithPathTraversalInFile.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare test file for be added to tar
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
func createTestTarFiveFiles(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarFiveFiles.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	for i := 0; i < 5; i++ {

		// prepare test file for be added to tar
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

func createTestTarGzWithEmptyFileName(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarEmptyNameDir.tar.gz")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare test file for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, "", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// addFileToTarArchive is a helper function to generate test content
func addFileToTarArchive(tarWriter *tar.Writer, fileName string, f1 *os.File) {

	if f1 == nil {

		// create a new dir/file header
		header := &tar.Header{
			Name:     fileName,
			Typeflag: tar.TypeDir,
		}

		// write the header
		if err := tarWriter.WriteHeader(header); err != nil {
			panic(err)
		}

		return
	}

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

// addFileToTarArchive is a helper function to generate test content
func addFifoToTarArchive(tarWriter *tar.Writer, fileName string, fifoPath string) {
	fileInfo, err := os.Lstat(fifoPath)
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
	header.Typeflag = tar.TypeFifo

	// write the header
	if err := tarWriter.WriteHeader(header); err != nil {
		panic(err)
	}

}

// addLinkToTarArchive is a helper function to generate test content
func addLinkToTarArchive(t *testing.T, tarWriter *tar.Writer, fileName string, linkTarget string) {
	// create a temporary dir for files in zip archive
	tmpDir := t.TempDir()

	// create dummy link to get data structure
	dummyLink := filepath.Join(tmpDir, "dummy-link")
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

// createTestTarWithAbsolutePathInFilename is a helper function to generate test content
func createTestTarWithAbsolutePathInFilename(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithMaliciousFilename.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare test file for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, "/absolute-path", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithAbsolutePathInFilenameWindows is a helper function to generate test content
func createTestTarWithAbsolutePathInFilenameWindows(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithMaliciousFilenameWindows.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare test file for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, "c:\\absolute-path", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}
