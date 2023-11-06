package extractor

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
		{
			name:           "malicous zip with zip slip attack",
			inputGenerator: createTestZipWithZipSlip,
			opts:           []config.ConfigOption{config.WithMaxExtractionTime(-1), config.WithContinueOnError(false)},
			expectError:    true,
		},
		{
			name:           "malicous zip with zip slip attack, but continue without error",
			inputGenerator: createTestZipWithZipSlip,
			opts:           []config.ConfigOption{config.WithMaxExtractionTime(-1), config.WithContinueOnError(true)},
			expectError:    false,
		},
		{
			name:           "malicous zip with zip slip attack, but follow sublinks",
			inputGenerator: createTestZipWithZipSlip,
			opts:           []config.ConfigOption{config.WithMaxExtractionTime(-1), config.WithFollowSymlinks(true)},
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

			unzipper := NewZip(config.NewConfig(tc.opts...))

			// perform actual tests
			input, _ := os.Open(tc.inputGenerator(testDir))
			want := tc.expectError
			err = unzipper.Unpack(context.Background(), input, testDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
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

// createTestZipWindows creates a test zip with windows file pathes file in dstDir for testing
func createTestZipWindows(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipWindows.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// prepare testfile for be added to zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// write file into zip
	w1, err := zipWriter.Create(`exampledir\foo\bar\test`)
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

// TestZipUnpackIllegalNames test with various testcases the implementation of zip.Unpack
func TestZipUnpackIllegalNames(t *testing.T) {

	// from: https://go.googlesource.com/go/+/refs/tags/go1.19.1/src/path/filepath/path_windows.go#19
	// from: https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names
	// removed `/` and `\` from tests, bc/ the zip lib cannot create directories as testfile
	var reservedNames []string
	var forbiddenCharacters []string

	if runtime.GOOS == "windows" {
		reservedNames = []string{
			"CON", "PRN", "AUX", "NUL",
			"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		}
		forbiddenCharacters = []string{`<`, `>`, `:`, `"`, `|`, `?`, `*`}
		for i := 0; i <= 31; i++ {
			fmt.Println(string(byte(i)))
			forbiddenCharacters = append(forbiddenCharacters, string(byte(i)))
		}
	} else {
		forbiddenCharacters = []string{"\x00"}
	}

	// test reserved names and forbidden chars
	unzipper := NewZip(config.NewConfig())
	for i, name := range append(reservedNames, forbiddenCharacters...) {
		t.Run(fmt.Sprintf("test %d %x", i, name), func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			// perform actual tests
			tFile := createTestZipWithCompressedFilename(testDir, name)
			input, _ := os.Open(tFile)
			// perform test
			err = unzipper.Unpack(context.Background(), input, testDir)
			if err == nil {
				t.Errorf("test case %d failed: test %s\n%s", i, name, err)
			}

		})

	}
}

// createTestZipWithCompressedFilename creates a test zip with compressedFilename as name in the archive in dstDir for testing
func createTestZipWithCompressedFilename(dstDir, compressedFilename string) string {

	targetFile := filepath.Join(dstDir, "ZipWithCompressedFilename.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// prepare testfile for be added to zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// write files with illegal names into zip

	w1, err := zipWriter.Create(compressedFilename)
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
	addLinkToZipArchive(zipWriter, "legitLinkName", "legitLinkTarget")

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
	addLinkToZipArchive(zipWriter, "../malicousLink", "nirvana")

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
	addLinkToZipArchive(zipWriter, "maliciousLink", "/etc/passwd")

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
	addLinkToZipArchive(zipWriter, "maliciousLink", "../malicousLinkTarget")

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// addLinkToZipArchive writes symlink linkName to linkTarget into zipWriter
func addLinkToZipArchive(zipWriter *zip.Writer, linkName string, linkTarget string) {

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
	if _, err := writer.Write([]byte(linkTarget)); err != nil {
		panic(err)
	}
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

// TestZipSuffix implements a test
func TestZipSuffix(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		unzip := NewZip(config.NewConfig())
		want := ".zip"
		got := unzip.FileSuffix()
		if got != want {
			t.Errorf("Unexpected filesuffix! want: %s, got :%s", want, got)
		}
	})
}

// TestZipOffset implements a test
func TestZipOffset(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		unzip := NewZip(config.NewConfig())
		want := 0
		got := unzip.Offset()
		if got != want {
			t.Errorf("Unexpected offset! want: %d, got :%d", want, got)
		}
	})
}

// TestZipSetConfig implements a test
func TestZipSetConfig(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		unzip := NewZip(config.NewConfig())
		newConfig := config.NewConfig()
		unzip.SetConfig(newConfig)

		// verify
		want := newConfig
		got := unzip.config
		if got != want {
			t.Errorf("Config not adjusted!")
		}
	})
}

// TestZipSetTarget implements a test
func TestZipSetTarget(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		unzip := NewZip(config.NewConfig())
		newTarget := target.NewOs()
		unzip.SetTarget(newTarget)

		// verify
		want := newTarget
		got := unzip.target
		if got != want {
			t.Errorf("Target not adjusted!")
		}
	})
}

// TestZipMagicBytes implements a test
func TestZipMagicBytes(t *testing.T) {
	t.Run("tc 0", func(t *testing.T) {
		// perform test
		untar := NewZip(config.NewConfig())
		want := [][]byte{
			{0x50, 0x4B, 0x03, 0x04},
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

// createTestTarWithSymlink is a helper function to generate test content
func createTestZipWithZipSlip(dstDir string) string {

	zipFile := filepath.Join(dstDir, "ZipWithZipSlip.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(zipFile)

	// add symlinks
	addLinkToZipArchive(zipWriter, "sub/to-parent", "../")
	addLinkToZipArchive(zipWriter, "sub/to-parent/one-above", "../")

	// close zip
	zipWriter.Close()

	// return path to zip
	return zipFile
}
