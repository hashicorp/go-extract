package extract_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/dsnet/compress/bzip2"
	"github.com/hashicorp/go-extract"
	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/internal/extractor"
	"github.com/hashicorp/go-extract/telemetry"
)

// TestGetUnpackFunction implements test cases
func TestGetUnpackFunction(t *testing.T) {
	// test cases
	cases := []struct {
		name           string
		createTestFile func(*testing.T, string) string
		expected       func(context.Context, extractor.Target, string, io.Reader, *config.Config) error
	}{
		{
			name:           "get zip extractor from file",
			createTestFile: createTestZip,
			expected:       extractor.UnpackZip,
		},
		{
			name:           "get tar extractor from file",
			createTestFile: createTestTar,
			expected:       extractor.UnpackTar,
		},
		{
			name:           "get gzip extractor from file",
			createTestFile: createTestGzipWithFile,
			expected:       extractor.UnpackGZip,
		},
		{
			name:           "get 7zip extractor for 7z file",
			createTestFile: create7zip,
			expected:       extractor.Unpack7Zip,
		},
		{
			name:           "get nil extractor fot textfile",
			createTestFile: createTestNonArchive,
			expected:       nil,
		},
	}

	// run cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			testDir := t.TempDir()

			// prepare vars
			want := tc.expected

			// perform actual tests
			f, err := os.Open(tc.createTestFile(t, testDir))
			if err != nil {
				f.Close()
				t.Fatal(err)
			}
			input, err := io.ReadAll(f)
			if err != nil {
				f.Close()
				t.Fatal(err)
			}
			got := extractor.AvailableExtractors.GetUnpackFunction(input, "")
			f.Close()

			wantT := reflect.TypeOf(want)
			gotT := reflect.TypeOf(got)

			if !gotT.AssignableTo(wantT) {
				t.Fatalf("\nexpected: %s\ngot: %s", wantT, gotT)
			}
		})
	}
}

// createGzip creates a gzip archive at dstFile with contents from input
func createGzip(dstFile string, input io.Reader) {
	// Create a new gzipped file
	gzippedFile, err := os.Create(dstFile)
	if err != nil {
		panic(err)
	}
	defer gzippedFile.Close()

	// Create a new gzip writer
	gzipWriter := gzip.NewWriter(gzippedFile)
	defer gzipWriter.Close()

	// Copy the contents of the original file to the gzip writer
	_, err = io.Copy(gzipWriter, input)
	if err != nil {
		panic(err)
	}

	// Flush the gzip writer to ensure all data is written
	gzipWriter.Flush()
}

func create7zip(t *testing.T, dstDir string) string {
	tmpFile := filepath.Join(t.TempDir(), "test.7z")
	archiveBytes, err := hex.DecodeString("377abcaf271c00049af18e7973000000000000002000000000000000a7e80f9801000b48656c6c6f20576f726c6421000000813307ae0fcef2b20c07c8437f41b1fafddb88b6d7636b8bd58a0e24a2f717a5f156e37f41fd00833298421d5d088c0cf987b30c0473663599e4d2f21cb69620038f10458109662135c3024189f42799abe3227b174a853e824f808b2efaab000017061001096300070b01000123030101055d001000000c760a015bcfa0a70000")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(tmpFile, archiveBytes, 0644); err != nil {
		t.Fatal(err)
	}
	return tmpFile
}

// createTestGzipWithFile creates a test gzip file in dstDir for testing
func createTestGzipWithFile(t *testing.T, dstDir string) string {

	// define target
	targetFile := filepath.Join(dstDir, "GzipWithFile.gz")

	// create a temporary dir for files in zip archive
	tmpDir := t.TempDir()

	// prepare test file for be added to zip
	testFilePath := filepath.Join(tmpDir, "test")
	createTestFile(testFilePath, "foobar content")
	f1, err := os.Open(testFilePath)
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	// create Gzip file
	createGzip(targetFile, f1)

	// return path to zip
	return targetFile
}

func createGzipFromFile(dstFile string, srcFile string) {
	// Create a new gzipped file
	gzippedFile, err := os.Create(dstFile)
	if err != nil {
		panic(err)
	}
	defer gzippedFile.Close()

	// Create a new gzip writer
	gzipWriter := gzip.NewWriter(gzippedFile)
	defer gzipWriter.Close()

	// open src file
	src, err := os.Open(srcFile)
	if err != nil {
		panic(err)
	}
	defer src.Close()

	// Copy the contents of the original file to the gzip writer
	_, err = io.Copy(gzipWriter, src)
	if err != nil {
		panic(err)
	}

	// Flush the gzip writer to ensure all data is written
	gzipWriter.Flush()
}

// createTestZip is a helper function to generate test data
func createTestZip(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TestZip.zip")

	// create a temporary dir for files in zip archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer
	archive, _ := os.Create(targetFile)
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	// prepare testfile for be added to zip
	testFilePath := filepath.Join(tmpDir, "test")
	createTestFile(testFilePath, "foobar content")
	f1, err := os.Open(testFilePath)
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	// write file into zip
	w1, _ := zipWriter.Create("test")
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	// return path to zip
	return targetFile
}

// createTestNonArchive is a helper function to generate test data
func createTestNonArchive(t *testing.T, dstDir string) string {
	targetFile := filepath.Join(dstDir, "test.txt")
	createTestFile(targetFile, "foo bar test")
	return targetFile
}

// createTestFile is a helper function to generate test files
func createTestFile(path string, content string) {
	err := createTestFileWithPerm(path, content, 0640)
	if err != nil {
		panic(err)
	}
}

// createTestTar is a helper function to generate test data
func createTestTar(t *testing.T, dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarNormal.tar")

	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// prepare generated zip+writer

	f, _ := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	tarWriter := tar.NewWriter(f)
	defer f.Close()

	// prepare testfile for be added to tar
	testFilePath := filepath.Join(tmpDir, "test")
	createTestFile(testFilePath, "foobar content")
	f1, err := os.Open(testFilePath)
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	// Add file to tar
	addFileToTarArchive(tarWriter, filepath.Base(f1.Name()), f1)

	// close tar
	tarWriter.Close()

	// return path to tar
	return targetFile
}

func createTestTarWithFiles(dst string, files map[string]string) {

	// prepare generated zip+writer
	f, _ := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	defer f.Close()
	tarWriter := tar.NewWriter(f)

	for nameInArchive, origFile := range files {
		f1, err := os.Open(origFile)
		if err != nil {
			panic(err)
		}
		defer f1.Close()

		addFileToTarArchive(tarWriter, nameInArchive, f1)
	}

	// close tar
	tarWriter.Close()
}

// addFileToTarArchive is a helper function
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

// TestUnpack is a test function
func TestUnpack(t *testing.T) {

	// test cases
	cases := []struct {
		name        string
		fn          func(*testing.T, string) string
		expectError bool
	}{
		{
			name:        "get zip extractor from file",
			fn:          createTestZip,
			expectError: false,
		},
		{
			name:        "get tar extractor from file",
			fn:          createTestTar,
			expectError: false,
		},
		{
			name:        "get gzip extractor from file",
			fn:          createTestGzipWithFile,
			expectError: false,
		},
		{
			name:        "get nil extractor fot textfile",
			fn:          createTestNonArchive,
			expectError: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// prepare vars
			want := tc.expectError

			// perform actual tests
			archive, err := os.Open(tc.fn(t, testDir))
			if err != nil {
				panic(err)
			}
			defer archive.Close()
			err = extract.Unpack(
				context.Background(),
				archive,
				testDir,
				config.NewConfig(
					config.WithOverwrite(true),
				),
			)
			got := err != nil

			// success if both are nil and no engine found
			if want != got {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, want, err)
			}
		})
	}
}

// TestUnpack is a test function
func TestUnpackToMemory(t *testing.T) {

	// test cases
	cases := []struct {
		name        string
		fn          func(*testing.T, string) string
		expectError bool
	}{
		{
			name:        "get zip extractor from file",
			fn:          createTestZip,
			expectError: false,
		},
		{
			name:        "get tar extractor from file",
			fn:          createTestTar,
			expectError: false,
		},
		{
			name:        "get gzip extractor from file",
			fn:          createTestGzipWithFile,
			expectError: false,
		},
		{
			name:        "get nil extractor fot textfile",
			fn:          createTestNonArchive,
			expectError: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// prepare vars
			want := tc.expectError

			// perform actual tests
			archive, err := os.Open(tc.fn(t, testDir))
			if err != nil {
				panic(err)
			}
			defer archive.Close()
			err = extract.UnpackTo(
				context.Background(),
				extractor.NewMemory(),
				"",
				archive,
				config.NewConfig(
					config.WithOverwrite(true),
				),
			)
			got := err != nil

			// success if both are nil and no engine found
			if want != got {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, want, err)
			}
		})
	}
}

func gen1024ByteGzip(t *testing.T, dstDir string) string {
	testFile := filepath.Join(dstDir, "GzipWithFile.gz")
	createGzip(testFile, strings.NewReader(strings.Repeat("A", 1024)))
	return testFile
}

func genSingleFileTar(t *testing.T, dstDir string) string {
	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// create test file
	testFile := filepath.Join(tmpDir, "testFile")
	createTestFile(testFile, strings.Repeat("A", 1024))

	tarFileName := filepath.Join(dstDir, "TarNormalSingleFile.tar")
	createTestTarWithFiles(tarFileName, map[string]string{"TestFile": testFile})
	return tarFileName
}

func genTarGzWith5Files(t *testing.T, dstDir string) string {
	// create a temporary dir for files in tar archive
	tmpDir := t.TempDir()

	// create test files
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("testFile%d", i))
		createTestFile(testFile, strings.Repeat("A", 1024))
	}
	tmpTar := filepath.Join(tmpDir, "tmp.tar")
	createTestTarWithFiles(tmpTar, map[string]string{
		"testFile0": filepath.Join(tmpDir, "testFile0"),
		"testFile1": filepath.Join(tmpDir, "testFile1"),
		"testFile2": filepath.Join(tmpDir, "testFile2"),
		"testFile3": filepath.Join(tmpDir, "testFile3"),
		"testFile4": filepath.Join(tmpDir, "testFile4"),
	})

	gzipFileName := filepath.Join(dstDir, "TarGzWith5Files.tar.gz")
	createGzipFromFile(gzipFileName, tmpTar)
	return gzipFileName
}

// TestTelemetryHook is a test function for the telemetry hook
func TestTelemetryHook(t *testing.T) {
	cases := []struct {
		name                  string
		inputGenerator        func(*testing.T, string) string
		inputName             string
		dst                   string
		WithContinueOnError   bool
		WithCreateDestination bool
		WithMaxExtractionSize int64
		WithMaxFiles          int64
		WithOverwrite         bool
		expectedTelemetryData telemetry.Data
		expectError           bool
	}{
		{
			name:                  "normal gzip with file",
			inputGenerator:        gen1024ByteGzip,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gz",
			},
			expectError: false,
		},
		{
			name:                  "normal gzip with file, and decompression target-name",
			inputGenerator:        gen1024ByteGzip,
			dst:                   "target-file", // important: the gzip decompression has a filename das dst
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gz",
			},
			expectError: false,
		},
		{
			name:                  "normal gzip with file and decompression target-name in sub-dir failing",
			inputGenerator:        gen1024ByteGzip,
			inputName:             "GzipWithFile.gz",
			dst:                   "sub/target", // important: the gzip decompression has a filename as dst
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 1,
				ExtractionSize:   0,
				ExtractedType:    "gz",
			},
			expectError: true,
		},
		{
			name:                  "normal gzip with file, and decompression target-name in sub-dir with sub-dir-creation",
			inputGenerator:        gen1024ByteGzip,
			inputName:             "GzipWithFile.gz",
			dst:                   "sub/target", // important: the gzip decompression has a filename das dst
			WithContinueOnError:   false,
			WithCreateDestination: true,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gz",
			},
			expectError: false,
		},
		{
			name:                  "normal tar with file",
			inputGenerator:        genSingleFileTar,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "tar",
			},
			expectError: false,
		},
		{
			name:                  "normal tar with file, extracted file too big",
			inputGenerator:        genSingleFileTar,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1023,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 1,
				ExtractionSize:   0,
				ExtractedType:    "tar",
			},
			expectError: true,
		},
		{
			name:                  "normal tar.gz with 5 files",
			inputGenerator:        genTarGzWith5Files,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the size(tar-archive) > 1025 * 5
			WithMaxFiles:          5,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   5,
				ExtractionErrors: 0,
				ExtractionSize:   1024 * 5,
				ExtractedType:    "tar.gz",
			},
			expectError: false,
		},
		{
			name:                  "normal tar.gz with file with max files limit",
			inputGenerator:        genTarGzWith5Files,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the size(tar-archive) > 1025 * 5
			WithMaxFiles:          4,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   4,
				ExtractionErrors: 1,
				ExtractionSize:   1024 * 4,
				ExtractedType:    "tar.gz",
			},
			expectError: true,
		},
		{
			name:                  "normal tar.gz with file failing bc/ of missing sub directory",
			inputGenerator:        genTarGzWith5Files,
			dst:                   "sub",
			WithContinueOnError:   true,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the size(tar-archive) > 1025 * 5
			WithMaxFiles:          5,
			WithOverwrite:         false,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 5,
				ExtractionSize:   0,
				ExtractedType:    "tar.gz",
			},
			expectError: false,
		},
		{
			name:                  "normal zip file",
			inputGenerator:        createTestZip,
			dst:                   ".",
			WithMaxFiles:          1,
			WithMaxExtractionSize: 14,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   14,
				ExtractedType:    "zip",
			},
			expectError: false,
		},
		{
			name:                  "normal zip file extraction size exceeded",
			inputGenerator:        createTestZip,
			dst:                   ".",
			WithMaxExtractionSize: 10,
			expectedTelemetryData: telemetry.Data{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 1,
				ExtractionSize:   0,
				ExtractedType:    "zip",
			},
			expectError: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// open file
			archive, err := os.Open(tc.inputGenerator(t, testDir))
			if err != nil {
				panic(err)
			}

			// prepare config
			var td *telemetry.Data
			hook := func(ctx context.Context, d *telemetry.Data) {
				td = d
			}

			cfg := config.NewConfig(
				config.WithContinueOnError(tc.WithContinueOnError),
				config.WithCreateDestination(tc.WithCreateDestination),
				config.WithMaxExtractionSize(tc.WithMaxExtractionSize),
				config.WithMaxFiles(tc.WithMaxFiles),
				config.WithOverwrite(tc.WithOverwrite),
				config.WithTelemetryHook(hook),
			)

			// perform actual tests
			ctx := context.Background()
			dstDir := filepath.Join(testDir, tc.dst)
			err = extract.Unpack(ctx, archive, dstDir, cfg)
			archive.Close()

			// check if error is expected
			if tc.expectError != (err != nil) {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, tc.expectError, err)
			}

			// compare collected and expected ExtractedFiles
			if td.ExtractedFiles != tc.expectedTelemetryData.ExtractedFiles {
				t.Errorf("test case %d failed: %s (ExtractedFiles)\nexpected: %v\ngot: %v\n%v", i, tc.name, tc.expectedTelemetryData.ExtractedFiles, td.ExtractedFiles, td)
			}

			// compare collected and expected ExtractionErrors
			if td.ExtractionErrors != tc.expectedTelemetryData.ExtractionErrors {
				t.Errorf("test case %d failed: %s (ExtractionErrors)\nexpected: %v\ngot: %v", i, tc.name, tc.expectedTelemetryData.ExtractionErrors, td.ExtractionErrors)
			}

			// compare collected and expected ExtractionSize
			if td.ExtractionSize != tc.expectedTelemetryData.ExtractionSize {
				t.Errorf("test case %d failed: %s (ExtractionSize [e:%v|g:%v])\nexpected: %v\ngot: %v", i, tc.name, tc.expectedTelemetryData.ExtractionSize, td.ExtractionSize, tc.expectedTelemetryData.ExtractionSize, td.ExtractionSize)
			}

			// compare collected and expected ExtractedType
			if td.ExtractedType != tc.expectedTelemetryData.ExtractedType {
				t.Errorf("test case %d failed: %s (ExtractedType)\nexpected: %v\ngot: %v", i, tc.name, tc.expectedTelemetryData.ExtractedType, td.ExtractedType)
			}

		})
	}
}

// TestUnpackWithTypes is a test function
func TestUnpackWithTypes(t *testing.T) {

	// test cases
	cases := []struct {
		name          string
		cfg           *config.Config
		archiveName   string
		content       []byte
		gen           func(target string, data []byte) io.Reader
		expectedFiles []string
		expectError   bool
	}{
		{
			name:          "get zip extractor from file",
			cfg:           config.NewConfig(config.WithExtractType(extractor.FileExtensionGZip)),
			archiveName:   "TestZip.gz",
			content:       compressGzip([]byte("foobar content")),
			gen:           createFile,
			expectedFiles: []string{"TestZip"},
			expectError:   false,
		},
		{
			name:        "set type to non-valid type and expect error",
			cfg:         config.NewConfig(config.WithExtractType("foo")),
			archiveName: "TestZip.gz",
			content:     compressGzip([]byte("foobar content")),
			gen:         createFile,
			expectError: true,
		},
		{
			name:          "get brotli extractor for file",
			cfg:           config.NewConfig(),
			archiveName:   "TestBrotli.br",
			content:       compressBrotli([]byte("foobar content")),
			gen:           createFile,
			expectedFiles: []string{"TestBrotli"},
			expectError:   false,
		},
		{
			name:        "extract zip file inside a tar.gz archive with extract type set to tar.gz",
			cfg:         config.NewConfig(config.WithExtractType(extractor.FileExtensionGZip)),
			archiveName: "example.json.zip.tar.gz",
			content: compressGzip(packTarWithContent([]tarContent{
				{
					Content:    packZipWithContent([]zipContent{{Name: "example.json", Content: []byte(`{"foo": "bar"}`)}}),
					Linktarget: "",
					Mode:       0644,
					Name:       "example.json.zip",
					Filetype:   tar.TypeReg,
				},
			})),
			gen:           createFile,
			expectedFiles: []string{"example.json.zip"},
			expectError:   false,
		},
		{
			name:        "extract zip file inside a tar.gz archive with extract type set to zip, so that it fails",
			cfg:         config.NewConfig(config.WithExtractType(extractor.FileExtensionZIP)),
			archiveName: "example.json.zip.tar.gz",
			content: compressGzip(packTarWithContent([]tarContent{
				{
					Content:    packZipWithContent([]zipContent{{Name: "example.json", Content: []byte(`{"foo": "bar"}`)}}),
					Linktarget: "",
					Mode:       0644,
					Name:       "example.json.zip",
					Filetype:   tar.TypeReg,
				},
			})),
			gen:         createFile,
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// perform actual tests
			archive := tc.gen(filepath.Join(testDir, tc.archiveName), tc.content)
			err := extract.Unpack(
				context.Background(),
				archive,
				testDir,
				tc.cfg,
			)
			defer func() {
				if closer, ok := archive.(io.Closer); ok {
					if closeErr := closer.Close(); closeErr != nil {
						t.Fatal(closeErr)
					}
				}
			}()

			if tc.expectError && err == nil {
				t.Errorf("\nexpected error\ngot: %s", err)
			}

			// check for created files
			for _, file := range tc.expectedFiles {
				_, err := os.Stat(filepath.Join(testDir, file))
				if err != nil {
					t.Errorf("\nexpected file: %s\ngot: %s", file, err)
				}
			}
		})
	}

}

// createFile creates a file with the given data and returns a reader for it.
func createFile(target string, data []byte) io.Reader {

	// Write the compressed data to the file
	if err := os.WriteFile(target, data, 0640); err != nil {
		panic(fmt.Errorf("error writing compressed data to file: %w", err))
	}

	// Open the file
	newFile, err := os.Open(target)
	if err != nil {
		panic(fmt.Errorf("error opening file: %w", err))
	}

	return newFile
}

// compressGzip compresses data using gzip algorithm
func compressGzip(data []byte) []byte {
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	if _, err := gzWriter.Write(data); err != nil {
		panic(err)
	}
	if err := gzWriter.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// Compress a byte slice with Brotli
func compressBrotli(data []byte) []byte {
	// Create a new Brotli writer
	brotliBuf := new(bytes.Buffer)
	brotliWriter := brotli.NewWriter(brotliBuf)

	// Write the data to the Brotli writer
	_, err := brotliWriter.Write(data)
	if err != nil {
		return nil
	}

	// Close the Brotli writer
	err = brotliWriter.Close()
	if err != nil {
		return nil
	}

	return brotliBuf.Bytes()
}

// compressBzip2 compresses data with bzip2 algorithm.
func compressBzip2(data []byte) []byte {
	// Create a new Bzip2 writer
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, &bzip2.WriterConfig{
		Level: bzip2.DefaultCompression,
	})
	if err != nil {
		panic(fmt.Errorf("error creating bzip2 writer: %w", err))
	}

	// Write the data to the Bzip2 writer
	_, err = w.Write(data)
	if err != nil {
		panic(fmt.Errorf("error writing data to bzip2 writer: %w", err))
	}

	// Close the Bzip2 writer
	err = w.Close()
	if err != nil {
		panic(fmt.Errorf("error closing bzip2 writer: %w", err))
	}

	return buf.Bytes()
}

// tarContent is a struct to store the content of a tar file
type tarContent struct {
	Content    []byte
	Linktarget string
	Mode       os.FileMode
	Name       string
	Filetype   byte
}

// packTarWithContent creates a tar file with the given content
func packTarWithContent(content []tarContent) []byte {

	// create tar writer
	writeBuffer := bytes.NewBuffer([]byte{})
	tw := tar.NewWriter(writeBuffer)

	// write content
	for _, c := range content {

		// create header
		hdr := &tar.Header{
			Name:     c.Name,
			Mode:     int64(c.Mode),
			Size:     int64(len(c.Content)),
			Linkname: c.Linktarget,
			Typeflag: c.Filetype,
		}

		// write header
		if err := tw.WriteHeader(hdr); err != nil {
			panic(err)
		}

		// write data
		if _, err := tw.Write(c.Content); err != nil {
			panic(err)
		}
	}

	// close tar writer
	if err := tw.Close(); err != nil {
		panic(err)
	}

	return writeBuffer.Bytes()
}

type zipContent struct {
	Name    string
	Content []byte
}

func packZipWithContent(content []zipContent) []byte {
	// create zip writer
	writeBuffer := bytes.NewBuffer([]byte{})
	zw := zip.NewWriter(writeBuffer)

	// write content
	for _, c := range content {

		// create header
		f, err := zw.Create(c.Name)
		if err != nil {
			panic(err)
		}

		// write data
		if _, err := f.Write(c.Content); err != nil {
			panic(err)
		}
	}

	// close zip writer
	if err := zw.Close(); err != nil {
		panic(err)
	}

	return writeBuffer.Bytes()
}

// TestUnsupportedArchiveNames is a test function
func TestUnsupportedArchiveNames(t *testing.T) {
	// test cases
	cases := []struct {
		name        string
		createInput func(string) string
		windows     string
		other       string
	}{
		{
			name: "valid archive name (gzip)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "test.gz"}, string(filepath.Separator))
				createTestFile(fPath, string(compressGzip([]byte("foobar content"))))
				return fPath
			},
			windows: "test",
			other:   "test",
		},
		{
			name: "invalid reported 1 (..bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "..bz2"}, string(filepath.Separator))
				createTestFile(fPath, string(compressBzip2([]byte("foobar content"))))
				return fPath
			},
			windows: "goextract-decompressed-content",
			other:   "goextract-decompressed-content",
		},
		{
			name: "invalid reported 2 (test..bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "test..bz2"}, string(filepath.Separator))
				createTestFile(fPath, string(compressBzip2([]byte("foobar content"))))
				return fPath
			},
			windows: "test.",
			other:   "test.",
		},
		{
			name: "invalid reported 3 (test.bz2.)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "test.bz2."}, string(filepath.Separator))
				createTestFile(fPath, string(compressBzip2([]byte("foobar content"))))
				return fPath
			},
			windows: "test.bz2..decompressed",
			other:   "test.bz2..decompressed",
		},
		{
			name: "invalid reported 4 (....bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, "....bz2"}, string(filepath.Separator))
				createTestFile(fPath, string(compressBzip2([]byte("foobar content"))))
				return fPath
			},
			windows: "goextract-decompressed-content",
			other:   "...",
		},
		{
			name: "invalid reported 5 (.. ..bz2)",
			createInput: func(path string) string {
				fPath := strings.Join([]string{path, ".. ..bz2"}, string(filepath.Separator))
				createTestFile(fPath, string(compressBzip2([]byte("foobar content"))))
				return fPath
			},
			windows: "goextract-decompressed-content",
			other:   ".. .",
		},
	}

	cfg := config.NewConfig(config.WithCreateDestination(true))

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// prepare file
			tmpDir := t.TempDir()
			tmpFile := tc.createInput(tmpDir)

			// run test
			archive, err := os.Open(tmpFile)
			if err != nil {
				t.Fatalf("error opening file: %s", err)
			}

			// perform actual tests
			ctx := context.Background()
			dstDir := filepath.Join(tmpDir, "out")
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				t.Fatalf("error creating directory: %s", err)
			}
			err = extract.Unpack(ctx, archive, dstDir, cfg)
			archive.Close()

			// check if error is expected
			if err != nil {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, false, err)
				return
			}

			// check for created files
			expectedFile := filepath.Join(tmpDir, "out", tc.other)
			if runtime.GOOS == "windows" {
				expectedFile = filepath.Join(tmpDir, "out", tc.windows)
			}
			if _, err := os.Stat(expectedFile); err != nil {
				t.Errorf("test case %d failed: %s\nexpected file: %s\ngot: %s", i, tc.name, expectedFile, err)
				return
			}

		})
	}

}

func TestWithCustomMode(t *testing.T) {

	umask := sniffUmask(t)

	tests := []struct {
		name        string
		data        []byte
		dst         string
		cfg         *config.Config
		expected    map[string]fs.FileMode
		expectError bool
	}{
		{
			name: "dir with 0755 and file with 0644",
			data: compressGzip(packTarWithContent([]tarContent{
				{
					Name: "sub/file",
					Mode: fs.FileMode(0644), // 420
				},
			})),
			cfg: config.NewConfig(
				config.WithCustomCreateDirMode(fs.FileMode(0755)), // 493
			),
			expected: map[string]fs.FileMode{
				"sub":      fs.FileMode(0755), // 493
				"sub/file": fs.FileMode(0644), // 420
			},
		},
		{
			name: "decompress with custom mode",
			data: compressGzip([]byte("foobar content")),
			dst:  "out", // specify decompressed file name
			cfg: config.NewConfig(
				config.WithCustomDecompressFileMode(fs.FileMode(0666)), // 438
			),
			expected: map[string]fs.FileMode{
				"out": fs.FileMode(0666), // 438
			},
		},
		{
			name:        "failing /bc of missing dir creation flag",
			data:        compressGzip([]byte("foobar content")),
			dst:         "foo/out", // specify decompressed file name in sub directory
			cfg:         config.NewConfig(),
			expected:    nil, // should error, bc/ missing dir creation flag
			expectError: true,
		},
		{
			name: "dir with 0755 and file with 0777",
			data: compressGzip([]byte("foobar content")),
			dst:  "foo/out",
			cfg: config.NewConfig(
				config.WithCreateDestination(true),                     // create destination^
				config.WithCustomCreateDirMode(fs.FileMode(0750)),      // 488
				config.WithCustomDecompressFileMode(fs.FileMode(0777)), // 511
			),
			expected: map[string]fs.FileMode{
				"foo":     fs.FileMode(0750), // 488
				"foo/out": fs.FileMode(0777), // 511
			},
			expectError: false, // because its just a compressed byte slice without any directories specified and WithCreateDestination is not set
		},
		{
			name: "dir with 0777 and file with 0777",
			data: compressGzip(packTarWithContent([]tarContent{
				{
					Name: "sub/file",
					Mode: fs.FileMode(0777), // 511
				},
			})),
			cfg: config.NewConfig(
				config.WithCustomCreateDirMode(fs.FileMode(0777)), // 511
			),
			expected: map[string]fs.FileMode{
				"sub":      fs.FileMode(0777), // 511
				"sub/file": fs.FileMode(0777), // 511
			},
		},
		{
			name: "file with 0000 permissions",
			data: compressGzip(packTarWithContent([]tarContent{
				{
					Name: "file",
					Mode: fs.FileMode(0000), // 0
				},
				{
					Name:     "dir/",
					Mode:     fs.FileMode(0000), // 0
					Filetype: tar.TypeDir,
				},
			})),
			cfg: config.NewConfig(),
			expected: map[string]fs.FileMode{
				"file": fs.FileMode(0000), // 0
				"dir":  fs.FileMode(0000), // 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// prepare test
			buf := bytes.NewBuffer(tt.data)
			ctx := context.Background()

			// create temp dir
			tmpDir := t.TempDir()
			dst := filepath.Join(tmpDir, tt.dst)

			// run test
			err := extract.Unpack(ctx, buf, dst, tt.cfg)
			if !tt.expectError && (err != nil) {
				t.Errorf("[%s] Expected no error, but got: %s", tt.name, err)
			}

			if tt.expectError && (err == nil) {
				t.Errorf("[%s] Expected error, but got none", tt.name)
			}

			// check results
			for name, expectedMode := range tt.expected {
				stat, err := os.Stat(filepath.Join(tmpDir, name))
				if err != nil {
					t.Errorf("[%s] Expected file %s to exist, but got: %s", tt.name, name, err)
				}

				skip := false
				// adjust for windows
				if runtime.GOOS == "windows" {
					skip = stat.IsDir() // ignore directories to be checked on windows, reason is that the mode is not under control of the go code
					expectedMode = toWindowsFileMode(stat.IsDir(), expectedMode)
				} else {
					// adjust for umask
					expectedMode = expectedMode & ^umask
				}

				if !skip && stat.Mode().Perm() != expectedMode.Perm() {
					t.Errorf("[%s] Expected directory/file '%s' to have mode %s, but got: %s", tt.name, name, expectedMode.Perm(), stat.Mode().Perm())
				}
			}
		})
	}
}

// sniffUmask is a helper function to get the umask
func sniffUmask(t *testing.T) fs.FileMode {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "file")

	// create 0777 file in temporary directory
	err := createTestFileWithPerm(tmpFile, "foobar content", 0777)
	if err != nil {
		t.Fatalf("error creating test file: %s", err)
	}

	// get stats
	stat, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("error getting file stats: %s", err)
	}

	// get umask
	umask := fs.FileMode(^stat.Mode().Perm() & 0777)

	// return the umask
	return umask
}

// toWindowsFileMode converts a fs.FileMode to a windows file mode
func toWindowsFileMode(isDir bool, mode fs.FileMode) fs.FileMode {

	// handle special case
	if isDir {
		return fs.FileMode(0777)
	}

	// check for write permission
	if mode&0200 != 0 {
		return fs.FileMode(0666)
	}

	// return the mode
	return fs.FileMode(0444)
}

// createTestFile is a helper function to generate test files
func createTestFileWithPerm(path string, content string, mode fs.FileMode) error {
	byteArray := []byte(content)
	return os.WriteFile(path, byteArray, mode)
}

func TestToWindowsFileMode(t *testing.T) {

	if runtime.GOOS != "windows" {
		t.Skip("skipping test on non-windows systems")
	}

	otherMasks := []int{00, 01, 02, 03, 04, 05, 06, 07}
	groupMasks := []int{00, 010, 020, 030, 040, 050, 060, 070}
	userMasks := []int{00, 0100, 0200, 0300, 0400, 0500, 0600, 0700}

	for _, dir := range []bool{true, false} {
		for _, o := range otherMasks {
			for _, g := range groupMasks {
				for _, u := range userMasks {

					// define test directory
					tmpDir := t.TempDir()
					fp := filepath.Join(tmpDir, "test")

					// define mode
					mode := fs.FileMode(u | g | o)

					// create test file or directory
					var err error
					if dir {
						err = os.MkdirAll(fp, mode)
					} else {
						err = createTestFileWithPerm(fp, "foobar content", mode)
					}
					if err != nil {
						t.Fatalf("error creating test file: %s", err)
					}

					// get stats
					stat, err := os.Stat(fp)
					if err != nil {
						t.Fatalf("error getting file stats: %s", err)
					}

					// calculate windows mode
					calculated := toWindowsFileMode(dir, mode)

					// check if the calculated mode is the same as the mode from the stat
					if stat.Mode().Perm() != calculated.Perm() {
						t.Errorf("toWindowsFileMode(%t, %s) calculated mode mode %s, but actual windows mode: %s", dir, mode, calculated.Perm(), stat.Mode().Perm())
					}

				}
			}
		}
	}
}
