package extract_test

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

	"github.com/hashicorp/go-extract"
)

var zipTests = []struct {
	name string

	testFileGenerator func(*testing.T, string) string
	opts              []extract.ConfigOption
	expectError       bool
}{
	{
		name:              "normal zip",
		testFileGenerator: createTestZipNormal,
		opts:              []extract.ConfigOption{},
		expectError:       false,
	},
	{
		name:              "normal zip, but pattern miss match",
		testFileGenerator: createTestZipNormal,
		opts:              []extract.ConfigOption{extract.WithPatterns("*foo")},
		expectError:       false,
	},
	{
		name:              "normal zip, cache in mem",
		testFileGenerator: createTestZipNormal,
		opts:              []extract.ConfigOption{extract.WithCacheInMemory(true)},
		expectError:       false,
	},
	{
		name:              "windows zip",
		testFileGenerator: createTestZipWindows,
		opts:              []extract.ConfigOption{},
		expectError:       false,
	},
	{
		name:              "normal zip with 5 files",
		testFileGenerator: createTestZipNormalFiveFiles,
		opts:              []extract.ConfigOption{},
		expectError:       false},
	{
		name:              "normal zip with 5 files",
		testFileGenerator: createTestZipNormalFiveFiles,
		opts:              []extract.ConfigOption{},
		expectError:       false,
	},
	{
		name:              "normal zip with 5 files, but extraction limit",
		testFileGenerator: createTestZipNormalFiveFiles,
		opts:              []extract.ConfigOption{extract.WithMaxFiles(1)},
		expectError:       true,
	},
	{
		name:              "zip with fifo (unix only)",
		testFileGenerator: createTestZipWithFifo,
		opts:              []extract.ConfigOption{},
		expectError:       true,
	},
	{
		name:              "zip with fifo, skip continue on error",
		testFileGenerator: createTestZipWithFifo,
		opts:              []extract.ConfigOption{extract.WithContinueOnError(true)},
		expectError:       false,
	},
	{
		name:              "zip with fifo, skip unsupported files",
		testFileGenerator: createTestZipWithFifo,
		opts:              []extract.ConfigOption{extract.WithContinueOnUnsupportedFiles(true)},
		expectError:       false,
	},
	{
		name:              "normal zip, but limited extraction size of 1 byte",
		testFileGenerator: createTestZipNormal,
		opts:              []extract.ConfigOption{extract.WithMaxExtractionSize(1)},
		expectError:       true,
	},
	{
		name:              "normal zip, but limited input size of 1 byte",
		testFileGenerator: createTestZipNormal,
		opts:              []extract.ConfigOption{extract.WithMaxInputSize(1)},
		expectError:       true,
	},
	{
		name:              "zip with dir traversal",
		testFileGenerator: createTestZipWithDirTraversal,
		opts:              []extract.ConfigOption{},
		expectError:       true,
	},
	{
		name:              "malicious zip with path traversal",
		testFileGenerator: createTestZipPathTraversal,
		opts:              []extract.ConfigOption{},
		expectError:       true,
	},
	{
		name:              "normal zip with symlink",
		testFileGenerator: createTestZipWithSymlink,
		opts:              []extract.ConfigOption{},
		expectError:       false,
	},
	{
		name:              "normal zip with symlink, but deny symlink extraction",
		testFileGenerator: createTestZipWithSymlink,
		opts:              []extract.ConfigOption{extract.WithDenySymlinkExtraction(true)},
		expectError:       true,
	},
	{
		name:              "normal zip with symlink, but deny symlink extraction, but continue without error",
		testFileGenerator: createTestZipWithSymlink,
		opts:              []extract.ConfigOption{extract.WithDenySymlinkExtraction(true), extract.WithContinueOnError(true)},
		expectError:       false,
	},
	{
		name:              "normal zip with symlink, but deny symlink extraction, but skip unsupported files",
		testFileGenerator: createTestZipWithSymlink,
		opts:              []extract.ConfigOption{extract.WithDenySymlinkExtraction(true), extract.WithContinueOnUnsupportedFiles(true)},
		expectError:       false,
	},
	{
		name:              "test max objects",
		testFileGenerator: createTestZipNormalFiveFiles,
		opts:              []extract.ConfigOption{extract.WithMaxFiles(1)},
		expectError:       true,
	},
	{
		name:              "malicious zip with symlink target containing path traversal",
		testFileGenerator: createTestZipWithSymlinkTargetPathTraversal,
		opts:              []extract.ConfigOption{},
		expectError:       true,
	},
	{
		name:              "malicious zip with symlink target referring absolute path",
		testFileGenerator: createTestZipWithSymlinkAbsolutePath,
		opts:              []extract.ConfigOption{},
		expectError:       runtime.GOOS != "windows",
	},
	{
		name:              "malicious zip with symlink name path traversal",
		testFileGenerator: createTestZipWithSymlinkPathTraversalName,
		opts:              []extract.ConfigOption{},
		expectError:       true,
	},
	{
		name:              "malicious zip with zip slip attack",
		testFileGenerator: createTestZipWithZipSlip,
		opts:              []extract.ConfigOption{extract.WithContinueOnError(false)},
		expectError:       true,
	},
	{
		name:              "malicious zip with zip slip attack, but continue without error",
		testFileGenerator: createTestZipWithZipSlip,
		opts:              []extract.ConfigOption{extract.WithContinueOnError(true)},
		expectError:       false,
	},
	{
		name:              "malicious zip with zip slip attack, but follow sub-links",
		testFileGenerator: createTestZipWithZipSlip,
		opts:              []extract.ConfigOption{extract.WithFollowSymlinks(true)},
		expectError:       false,
	},
	{
		name:              "file thats not zip",
		testFileGenerator: generateRandomFile,
		opts:              []extract.ConfigOption{},
		expectError:       true,
	},
}

func TestUnpackZip_file(t *testing.T) {
	// run cases with read from disk
	for _, test := range zipTests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new target
			testingTarget := extract.NewDisk()

			// create testing directory
			testDir := t.TempDir()

			// perform actual tests
			input, err := os.Open(test.testFileGenerator(t, testDir))
			if err != nil {
				t.Errorf("cannot open file: %s", err)
			}
			defer input.Close()
			want := test.expectError
			err = extract.UnpackZip(context.Background(), testingTarget, testDir, input, extract.NewConfig(test.opts...))
			got := err != nil
			if got != want {
				t.Error(err)
				defer input.Close()
			}
		})
	}
}

func TestUnpackZip_mem(t *testing.T) {
	for _, test := range zipTests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new target
			testingTarget := extract.NewDisk()

			// create testing directory
			testDir := t.TempDir()

			// perform actual tests
			var buf bytes.Buffer
			input, err := os.Open(test.testFileGenerator(t, testDir))
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()

			if _, err := io.Copy(&buf, input); err != nil {
				t.Error(err.Error())
			}

			want := test.expectError

			err = extract.UnpackZip(context.Background(), testingTarget, testDir, &buf, extract.NewConfig(test.opts...))
			got := err != nil
			if got != want {
				t.Fatal(err)
			}
		})
	}

}

// TestZipUnpack_seeker test with various test cases the implementation of zip.Unpack
func TestZipUnpack_seeker(t *testing.T) {
	for _, test := range zipTests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new target
			testingTarget := extract.NewDisk()

			// create testing directory
			testDir := t.TempDir()
			var buf bytes.Buffer
			input, err := os.Open(test.testFileGenerator(t, testDir))
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			if _, err := io.Copy(&buf, input); err != nil {
				t.Error(err.Error())
			}
			// to readerAt
			br := bytes.NewReader(buf.Bytes())

			// perform actual tests
			want := test.expectError
			err = extract.UnpackZip(context.Background(), testingTarget, testDir, br, extract.NewConfig(test.opts...))
			got := err != nil
			if got != want {
				t.Error(err)
			}
		})
	}

}

func generateRandomFile(t *testing.T, testDir string) string {
	targetFile := filepath.Join(testDir, "randomFile")
	createTestFile(t, targetFile, "foobar content")
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := extract.IsZip(test.data); got != test.want {
				t.Errorf("IsZip() = %v, want %v", got, test.want)
			}
		})
	}
}

// createTestZipNormal creates a test zip file in dstDir for testing
func createTestZipNormal(t *testing.T, dstDir string) string {
	t.Helper()
	p := packZip(t, []archiveContent{
		{Name: "test", Content: []byte("foobar content"), Mode: 0640, Filetype: 0},
		{Name: "sub/", Mode: 0755, Filetype: uint32(fs.ModeDir)},
	})
	targetFile := filepath.Join(dstDir, "ZipNormal.zip")
	if err := os.WriteFile(targetFile, p, 0644); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// createTestZipWithDirTraversal creates a test zip file with a directory in dstDir for testing
func createTestZipWithDirTraversal(t *testing.T, dstDir string) string {
	t.Helper()

	targetFile := filepath.Join(dstDir, "ZipWithDir.zip")

	// prepare generated zip+writer
	f, zipWriter := createZip(t, targetFile)
	defer f.Close()

	// create directory in zip
	_, err := zipWriter.Create("sub/../../outside/")
	if err != nil {
		t.Fatal(err)
	}

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// createTestZipWindows creates a test zip with windows-style file paths file in dstDir for testing
func createTestZipWindows(t *testing.T, dstDir string) string {
	t.Helper()
	p := packZip(t, []archiveContent{
		{Name: `example-dir\foo\bar\test`, Content: []byte("foobar content"), Mode: 0640, Filetype: 0},
	})
	targetFile := filepath.Join(dstDir, "ZipWindows.zip")
	if err := os.WriteFile(targetFile, p, 0644); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// TestZipUnpackIllegalNames tests, with various cases, the implementation of zip.Unpack
func TestZipUnpackIllegalNames(t *testing.T) {
	t.Helper()

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

	tests := append(reservedNames, forbiddenCharacters...)

	// test reserved names and forbidden chars
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a new target
			testingTarget := extract.NewDisk()

			// create testing directory
			testDir := t.TempDir()

			// perform actual tests
			tFile := createTestZipWithCompressedFilename(t, testDir, name)
			input, err := os.Open(tFile)
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()

			// perform test
			err = extract.UnpackZip(context.Background(), testingTarget, testDir, input, extract.NewConfig())
			if err == nil {
				t.Error(err)
			}
		})

	}
}

// createTestZipWithCompressedFilename creates a test zip with the name 'ZipWithCompressedFilename.zip' in
// dstDir with filenameInsideTheArchive as name for the file inside the archive.
func createTestZipWithCompressedFilename(t *testing.T, dstDir, filenameInsideTheArchive string) string {
	t.Helper()
	p := packZip(t, []archiveContent{
		{Content: []byte("foobar content"), Name: filenameInsideTheArchive, Mode: 0640, Filetype: 0},
		{Name: "sub/", Mode: 0755, Filetype: uint32(fs.ModeDir)},
	})
	targetFile := filepath.Join(dstDir, "ZipWithCompressedFilename.zip")
	if err := os.WriteFile(targetFile, p, 0644); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// createTestZipPathTraversal creates a test with a filename path traversal zip file in dstDir for testing
func createTestZipPathTraversal(t *testing.T, dstDir string) string {
	t.Helper()
	p := packZip(t, []archiveContent{
		{Name: "../test", Content: []byte("foobar content"), Mode: 0640, Filetype: 0},
	})
	targetFile := filepath.Join(dstDir, "ZipTraversal.zip")
	if err := os.WriteFile(targetFile, p, 0644); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// createTestZipNormalFiveFiles creates a test zip file with five files in dstDir for testing
func createTestZipNormalFiveFiles(t *testing.T, dstDir string) string {
	t.Helper()
	var archiveContents []archiveContent
	for i := 0; i < 5; i++ {
		archiveContents = append(archiveContents, archiveContent{
			Content:  []byte("foobar content"),
			Name:     fmt.Sprintf("test%d", i),
			Mode:     0640,
			Filetype: 0,
		})
	}
	p := packZip(t, archiveContents)
	targetFile := filepath.Join(dstDir, "ZipNormalFiveFiles.zip")
	if err := os.WriteFile(targetFile, p, 0644); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// createTestZipWithSymlink creates a test zip file with a legit sym link in dstDir for testing
func createTestZipWithSymlink(t *testing.T, dstDir string) string {
	t.Helper()

	targetFile := filepath.Join(dstDir, "ZipNormalWithSymlink.zip")

	// prepare generated zip+writer
	f, zipWriter := createZip(t, targetFile)
	defer f.Close()

	// add link to archive
	addLinkToZipArchive(t, zipWriter, "legitLinkName", "legitLinkTarget")

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// createTestZipWithSymlinkPathTraversalName creates a test zip file, with a symlink, which filename contains a path traversal, in dstDir for testing
func createTestZipWithSymlinkPathTraversalName(t *testing.T, dstDir string) string {
	t.Helper()

	targetFile := filepath.Join(dstDir, "createTestZipWithSymlinkPathTraversalName.zip")

	// prepare generated zip+writer
	f, zipWriter := createZip(t, targetFile)
	defer f.Close()

	// add link to archive
	addLinkToZipArchive(t, zipWriter, "../maliciousLink", "nirvana")

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// createTestZipWithSymlinkAbsolutePath creates a test zip file, with a symlink to a absolute path, in dstDir for testing
func createTestZipWithSymlinkAbsolutePath(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "ZipWithSymlinkTargetAbsolutePath.zip")

	// prepare generated zip+writer
	f, zipWriter := createZip(t, targetFile)
	defer f.Close()

	// add link to archive
	addLinkToZipArchive(t, zipWriter, "maliciousLink", "/etc/passwd")

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// createTestZipWithSymlinkTargetPathTraversal creates a test zip file, with a path traversal in the link target, in dstDir for testing
func createTestZipWithSymlinkTargetPathTraversal(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "ZipWithSymlinkTargetPathTraversal.zip")

	// prepare generated zip+writer
	f, zipWriter := createZip(t, targetFile)
	defer f.Close()

	// add link to archive
	addLinkToZipArchive(t, zipWriter, "maliciousLink", "../maliciousLinkTarget")

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// addLinkToZipArchive writes symlink linkName to linkTarget into zipWriter
func addLinkToZipArchive(t *testing.T, zipWriter *zip.Writer, linkName string, linkTarget string) {
	// create a temporary dir for files in zip archive
	tmpDir := t.TempDir()

	// create dummy link to get data structure
	dummyLink := filepath.Join(tmpDir, "dummy-link")
	if err := os.Symlink("nirvana", dummyLink); err != nil {
		t.Fatal(err)
	}

	// get file stats for testing operating system
	info, err := os.Lstat(dummyLink)
	if err != nil {
		t.Fatal(err)
	}

	// get file header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		t.Fatal(err)
	}

	// adjust file headers
	header.Name = linkName
	header.Method = zip.Deflate

	// create writer for link
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}

	// Write symlink's target to writer - file's body for symlinks is the symlink
	if _, err := writer.Write([]byte(linkTarget)); err != nil {
		t.Fatal(err)
	}
}

// createTestZipWithFifo creates a test zip file with a fifo file in dstDir for testing
func createTestZipWithFifo(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "ZipWithFifo.zip")

	// prepare generated zip+writer
	f, zipWriter := createZip(t, targetFile)
	defer f.Close()

	// add fifo to archive
	addFifoToZipArchive(t, zipWriter)

	// close zip
	zipWriter.Close()

	// return path to zip
	return targetFile
}

// AddFifoToZipArchive writes fifo into zipWriter
func addFifoToZipArchive(t *testing.T, zipWriter *zip.Writer) {
	// create a temporary dir for files in zip archive
	tmpDir := t.TempDir()

	// create dummy fifo to get data structure
	tmpFile, err := os.CreateTemp(tmpDir, "fifo")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	info, err := os.Lstat(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// get file header
	header, err := zip.FileInfoHeader(info)
	header.SetMode(fs.ModeDevice)
	if err != nil {
		t.Fatal(err)
	}

	// adjust file headers
	header.Name = "fifo"
	header.Method = zip.Deflate

	// create writer for fifo
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}

	// Write fifo's target to writer - file's body for fifos is the fifo
	if _, err := writer.Write([]byte("nirvana")); err != nil {
		t.Fatal(err)
	}
}

// createZip creates a new zip file in filePath
func createZip(t *testing.T, filePath string) (*os.File, *zip.Writer) {
	targetFile := filepath.Join(filePath)
	archive, err := os.Create(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	return archive, zip.NewWriter(archive)
}

// createTestTarWithSymlink is a helper function to generate test content
func createTestZipWithZipSlip(t *testing.T, dstDir string) string {
	zipFile := filepath.Join(dstDir, "ZipWithZipSlip.tar")

	// prepare generated zip+writer
	f, zipWriter := createZip(t, zipFile)
	defer f.Close()

	// add symlinks
	addLinkToZipArchive(t, zipWriter, "sub/to-parent", "../")
	addLinkToZipArchive(t, zipWriter, "sub/to-parent/one-above", "../")

	// close zip
	zipWriter.Close()

	// return path to zip
	return zipFile
}
