package extractor

import (
	"archive/tar"
	"bytes"
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
			opts:           []config.ConfigOption{config.WithVerbose(true)},
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
		{
			name:           "malicous tar.gz with empty name for a dir",
			inputGenerator: createTestTarGzWithEmptyNameDirectory,
			opts:           []config.ConfigOption{},
			expectError:    true,
		},
		{
			name:           "malicous tar with .. as filename",
			inputGenerator: createTestTarDotDotFilename,
			opts:           []config.ConfigOption{config.WithOverwrite(true)},
			expectError:    true,
		},
		{
			name:           "malicous tar with FIFIO filetype",
			inputGenerator: createTestTarWithFIFO,
			opts:           []config.ConfigOption{config.WithOverwrite(true)},
			expectError:    true,
		},
		{
			name:           "malicous tar with zip slip attack",
			inputGenerator: createTestTarWithZipSlip,
			opts:           []config.ConfigOption{config.WithMaxExtractionTime(-1), config.WithContinueOnError(false)},
			expectError:    true,
		}, {
			name:           "malicious tar with absolut path in filename",
			inputGenerator: createTestTarWithMaliciousFilename,
			opts:           []config.ConfigOption{config.WithVerbose(true)},
			expectError:    true,
		}, {
			name:           "malicious tar with absolut path in filename (windows)",
			inputGenerator: createTestTarWithMaliciousFilenameWindows,
			opts:           []config.ConfigOption{config.WithVerbose(true)},
			expectError:    true,
		}, {
			name:           "malicious tar with absolut path in filename, but continue",
			inputGenerator: createTestTarWithMaliciousFilename,
			opts:           []config.ConfigOption{config.WithContinueOnError(true)},
			expectError:    false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("test%d_*", i))
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
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
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
	addFileToTarArchive(tarWriter, "test", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

func createTestTarWithFIFO(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithFIFO.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

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
func createTestTarDotDotFilename(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarDotDot.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare testfile for be added to tar
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

// createTestTarWithSymlink is a helper function to generate test content
func createTestTarWithZipSlip(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithZipSlip.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// add symlinks
	addLinkToTarArchive(tarWriter, "sub/to-parent", "../")
	addLinkToTarArchive(tarWriter, "sub/to-parent/one-above", "../")

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

func createTestTarGzWithEmptyNameDirectory(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarEmptyNameDir.tar.gz")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare testfile for be added to tar
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

// TestTarSuffix implements a test
func TestTarSuffix(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		untar := NewTar(config.NewConfig())
		want := ".tar"
		got := untar.FileSuffix()
		if got != want {
			t.Errorf("Unexpected filesuffix! want: %s, got :%s", want, got)
		}
	})
}

// TestTarOffset implements a test
func TestTarOffset(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		untar := NewTar(config.NewConfig())
		want := 257
		got := untar.Offset()
		if got != want {
			t.Errorf("Unexpected offset! want: %d, got :%d", want, got)
		}
	})
}

// TestTarSetConfig implements a test
func TestTarSetConfig(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		untar := NewTar(config.NewConfig())
		newConfig := config.NewConfig()
		untar.SetConfig(newConfig)

		// verify
		want := newConfig
		got := untar.config
		if got != want {
			t.Errorf("Config not adjusted!")
		}
	})
}

// TestTarSetTarget implements a test
func TestTarSetTarget(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		untar := NewTar(config.NewConfig())
		newTarget := target.NewOs()
		untar.SetTarget(newTarget)

		// verify
		want := newTarget
		got := untar.target
		if got != want {
			t.Errorf("Target not adjusted!")
		}
	})
}

// TestTarMagicBytes implements a test
func TestTarMagicBytes(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		untar := NewTar(config.NewConfig())
		want := [][]byte{
			{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
			{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x20, 0x00},
		}
		got := untar.magicBytes
		for idx := range got {
			for idy := range got[idx] {
				if got[idx][idy] != want[idx][idy] {
					t.Errorf("Magic byte missmatche!")
				}
			}
		}
	})
}

// TestTarMagicBytes implements a test
func TestTarMagicBytesMatch(t *testing.T) {

	cases := []struct {
		name        string
		input       []byte
		expectMatch bool
	}{
		{
			name:        "normal tar header",
			input:       []byte{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
			expectMatch: true,
		},
		{
			name:        "empty tar header",
			input:       []byte{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
			expectMatch: true,
		},
		{
			name:        "tar header missmatch",
			input:       []byte{0xaa, 0xbb, 0xcc, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			expectMatch: false,
		},
		{
			name:        "too short tar header",
			input:       []byte{0xFF},
			expectMatch: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			untar := NewTar(config.NewConfig())

			// prepare testdata
			byteBuffer := &bytes.Buffer{}
			for i := 0; i < untar.Offset(); i++ {
				byteBuffer.WriteByte(0x00)
			}
			byteBuffer.Write(tc.input)

			// perform test
			want := tc.expectMatch
			got := untar.MagicBytesMatch(byteBuffer.Bytes())

			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}

		})
	}

}

// createTestTarWithMaliciousFilename is a helper function to generate test content
func createTestTarWithMaliciousFilename(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithMaliciousFilename.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare testfile for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, "/absolut-path", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}

// createTestTarWithMaliciousFilename is a helper function to generate test content
func createTestTarWithMaliciousFilenameWindows(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarWithMaliciousFilenameWindows.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	tarWriter := createTar(targetFile)

	// prepare testfile for be added to tar
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, "c:\\absolut-path", f1)

	// close zip
	tarWriter.Close()

	// return path to zip
	return targetFile
}
