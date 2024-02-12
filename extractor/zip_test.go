package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestZipUnpack test with various test cases the implementation of zip.Unpack
func TestZipUnpack(t *testing.T) {
	cases := []struct {
		name              string
		testFileGenerator func(string) string
		opts              []config.ConfigOption
		expectError       bool
	}{
		{
			name:              "normal zip",
			testFileGenerator: createTestZipNormal,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "normal zip, but pattern miss match",
			testFileGenerator: createTestZipNormal,
			opts:              []config.ConfigOption{config.WithPattern("*foo")},
			expectError:       false,
		},
		{
			name:              "normal zip, cache in mem",
			testFileGenerator: createTestZipNormal,
			opts:              []config.ConfigOption{config.WithCacheInMemory(true)},
			expectError:       false,
		},
		{
			name:              "windows zip",
			testFileGenerator: createTestZipWindows,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "normal zip with 5 files",
			testFileGenerator: createTestZipNormalFiveFiles,
			opts:              []config.ConfigOption{},
			expectError:       false},
		{
			name:              "normal zip with 5 files",
			testFileGenerator: createTestZipNormalFiveFiles,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "normal zip with 5 files, but extraction limit",
			testFileGenerator: createTestZipNormalFiveFiles,
			opts:              []config.ConfigOption{config.WithMaxFiles(1)},
			expectError:       true,
		},
		{
			name:              "zip with fifo (unix only)",
			testFileGenerator: createTestZipWithFifo,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "zip with fifo, skip continue on error",
			testFileGenerator: createTestZipWithFifo,
			opts:              []config.ConfigOption{config.WithContinueOnError(true)},
			expectError:       false,
		},
		{
			name:              "zip with fifo, skip unsupported files",
			testFileGenerator: createTestZipWithFifo,
			opts:              []config.ConfigOption{config.WithContinueOnUnsupportedFiles(true)},
			expectError:       false,
		},
		{
			name:              "normal zip, but limited extraction size of 1 byte",
			testFileGenerator: createTestZipNormal,
			opts:              []config.ConfigOption{config.WithMaxExtractionSize(1)},
			expectError:       true,
		},
		{
			name:              "normal zip, but limited input size of 1 byte",
			testFileGenerator: createTestZipNormal,
			opts:              []config.ConfigOption{config.WithMaxInputSize(1)},
			expectError:       true,
		},
		{
			name:              "zip with dir traversal",
			testFileGenerator: createTestZipWithDirTraversal,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "malicious zip with path traversal",
			testFileGenerator: createTestZipPathTraversal,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "normal zip with symlink",
			testFileGenerator: createTestZipWithSymlink,
			opts:              []config.ConfigOption{},
			expectError:       false,
		},
		{
			name:              "normal zip with symlink, but deny symlink extraction",
			testFileGenerator: createTestZipWithSymlink,
			opts:              []config.ConfigOption{config.WithAllowSymlinks(false)},
			expectError:       true,
		},
		{
			name:              "normal zip with symlink, but deny symlink extraction, but continue without error",
			testFileGenerator: createTestZipWithSymlink,
			opts:              []config.ConfigOption{config.WithAllowSymlinks(false), config.WithContinueOnError(true)},
			expectError:       false,
		},
		{
			name:              "normal zip with symlink, but deny symlink extraction, but skip unsupported files",
			testFileGenerator: createTestZipWithSymlink,
			opts:              []config.ConfigOption{config.WithAllowSymlinks(false), config.WithContinueOnUnsupportedFiles(true)},
			expectError:       false,
		},
		{
			name:              "test max objects",
			testFileGenerator: createTestZipNormalFiveFiles,
			opts:              []config.ConfigOption{config.WithMaxFiles(1)},
			expectError:       true,
		},
		{
			name:              "malicious zip with symlink target containing path traversal",
			testFileGenerator: createTestZipWithSymlinkTargetPathTraversal,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "malicious zip with symlink target referring absolute path",
			testFileGenerator: createTestZipWithSymlinkAbsolutePath,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "malicious zip with symlink name path traversal",
			testFileGenerator: createTestZipWithSymlinkPathTraversalName,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
		{
			name:              "malicious zip with zip slip attack",
			testFileGenerator: createTestZipWithZipSlip,
			opts:              []config.ConfigOption{config.WithContinueOnError(false)},
			expectError:       true,
		},
		{
			name:              "malicious zip with zip slip attack, but continue without error",
			testFileGenerator: createTestZipWithZipSlip,
			opts:              []config.ConfigOption{config.WithContinueOnError(true)},
			expectError:       false,
		},
		{
			name:              "malicious zip with zip slip attack, but follow sub-links",
			testFileGenerator: createTestZipWithZipSlip,
			opts:              []config.ConfigOption{config.WithFollowSymlinks(true)},
			expectError:       false,
		},
		{
			name:              "file thats not zip",
			testFileGenerator: generateRandomFile,
			opts:              []config.ConfigOption{},
			expectError:       true,
		},
	}

	// run cases with read from disk
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			unziper := NewZip()

			// perform actual tests
			input, _ := os.Open(tc.testFileGenerator(testDir))
			want := tc.expectError
			err = unziper.Unpack(context.Background(), input, testDir, target.NewOS(), config.NewConfig(tc.opts...))
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}

	// run cases with read from memory
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			unziper := NewZip()

			// perform actual tests
			var buf bytes.Buffer
			input, _ := os.Open(tc.testFileGenerator(testDir))
			if _, err := io.Copy(&buf, input); err != nil {
				t.Errorf(err.Error())
			}
			want := tc.expectError
			err = unziper.Unpack(context.Background(), &buf, testDir, target.NewOS(), config.NewConfig(tc.opts...))
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}

}

func generateRandomFile(testDir string) string {
	targetFile := filepath.Join(testDir, "randomFile")
	createTestFile(targetFile, "foobar content")
	return targetFile
}

func TestIsZip(t *testing.T) {
	zipBytes := []byte{0x50, 0x4B, 0x03, 0x04}    // Magic bytes for ZIP files
	nonZipBytes := []byte{0x01, 0x02, 0x03, 0x04} // Random bytes

	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "ZIP bytes",
			data: zipBytes,
			want: true,
		},
		{
			name: "Non-ZIP bytes",
			data: nonZipBytes,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsZip(tt.data); got != tt.want {
				t.Errorf("IsZip() = %v, want %v", got, tt.want)
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

	// prepare test file for be added to zip
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

	// create directory in zip
	_, err = zipWriter.Create("sub/")
	if err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// createTestZipWithDirTraversal creates a test zip file with a directory in dstDir for testing
func createTestZipWithDirTraversal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipWithDir.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// create directory in zip
	_, err := zipWriter.Create("sub/../../outside/")
	if err != nil {
		panic(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// createTestZipWindows creates a test zip with windows-style file paths file in dstDir for testing
func createTestZipWindows(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipWindows.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// prepare test file that will be added to the zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	// write file into zip
	w1, err := zipWriter.Create(`example-dir\foo\bar\test`)
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

// TestZipUnpackIllegalNames tests, with various cases, the implementation of zip.Unpack
func TestZipUnpackIllegalNames(t *testing.T) {

	// from: https://go.googlesource.com/go/+/refs/tags/go1.19.1/src/path/filepath/path_windows.go#19
	// from: https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names
	// removed `/` and `\` from tests, bc/ the zip lib cannot create directories as test file
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
	unziper := NewZip()
	unzipTarget := target.NewOS()
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
			err = unziper.Unpack(context.Background(), input, testDir, unzipTarget, config.NewConfig())
			if err == nil {
				t.Errorf("test case %d failed: test %s\n%s", i, name, err)
			}

		})

	}
}

// createTestZipWithCompressedFilename creates a test zip with the name 'ZipWithCompressedFilename.zip' in
// dstDir with filenameInsideTheArchive as name for the file inside the archive.
func createTestZipWithCompressedFilename(dstDir, filenameInsideTheArchive string) string {

	targetFile := filepath.Join(dstDir, "ZipWithCompressedFilename.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// prepare test file for be added to zip
	f1 := createTestFile(filepath.Join(tmpDir, "test"), "foobar content")
	defer f1.Close()

	w1, err := zipWriter.Create(filenameInsideTheArchive)
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

// createTestZipPathTraversal creates a test with a filename path traversal zip file in dstDir for testing
func createTestZipPathTraversal(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipTraversal.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// prepare test file for be added to zip
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
		// prepare test file for be added to zip
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
	addLinkToZipArchive(zipWriter, "../maliciousLink", "nirvana")

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// createTestZipWithSymlinkAbsolutePath creates a test zip file, with a symlink to a absolute path, in dstDir for testing
func createTestZipWithSymlinkAbsolutePath(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipWithSymlinkTargetAbsolutePath.zip")

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
	addLinkToZipArchive(zipWriter, "maliciousLink", "../maliciousLinkTarget")

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
	dummyLink := filepath.Join(tmpDir, "dummy-link")
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

// createTestZipWithFifo creates a test zip file with a fifo file in dstDir for testing
func createTestZipWithFifo(dstDir string) string {

	targetFile := filepath.Join(dstDir, "ZipWithFifo.zip")

	// prepare generated zip+writer
	zipWriter := createZip(targetFile)

	// add fifo to archive
	addFifoToZipArchive(zipWriter)

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// AddFifoToZipArchive writes fifo into zipWriter
func addFifoToZipArchive(zipWriter *zip.Writer) {

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// create dummy fifo to get data structure
	tmpFile, err := os.CreateTemp(tmpDir, "fifo")
	if err != nil {
		panic(err)
	}
	info, err := os.Lstat(tmpFile.Name())
	if err != nil {
		panic(err)
	}

	// get file header
	header, err := zip.FileInfoHeader(info)
	header.SetMode(fs.ModeDevice)
	if err != nil {
		panic(err)
	}

	// adjust file headers
	header.Name = "fifo"
	header.Method = zip.Deflate

	// create writer for fifo
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		panic(err)
	}

	// Write fifo's target to writer - file's body for fifos is the fifo target.
	if _, err := writer.Write([]byte("nirvana")); err != nil {
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
