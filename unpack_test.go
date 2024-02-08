package extract

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
	"github.com/hashicorp/go-extract/target"
)

// TestFindExtractor implements test cases
func TestFindExtractor(t *testing.T) {
	// test cases
	cases := []struct {
		name           string
		createTestFile func(string) string
		expected       Extractor
	}{
		{
			name:           "get zip extractor from file",
			createTestFile: createTestZip,
			expected:       extractor.NewZip(),
		},
		{
			name:           "get tar extractor from file",
			createTestFile: createTestTar,
			expected:       extractor.NewTar(),
		},
		{
			name:           "get gzip extractor from file",
			createTestFile: createTestGzipWithFile,
			expected:       extractor.NewGzip(),
		},
		{
			name:           "get nil extractor fot textfile",
			createTestFile: createTestNonArchive,
			expected:       nil,
		},
	}

	// create testing directory
	testDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		t.Errorf(err.Error())
	}
	testDir = filepath.Clean(testDir) + string(os.PathSeparator)
	defer os.RemoveAll(testDir)

	// run cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// prepare vars
			want := tc.expected

			// perform actual tests
			f, err := os.Open(tc.createTestFile(testDir))
			if err != nil {
				t.Fatal(err)
			}
			input, err := io.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			got := findExtractor(input)

			// success if both are nil and no engine found
			if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", want) {
				t.Fatalf("expected: %v\ngot: %v", want, got)
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

// createTestGzipWithFile creates a test gzip file in dstDir for testing
func createTestGzipWithFile(dstDir string) string {

	// define target
	targetFile := filepath.Join(dstDir, "GzipWithFile.gz")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

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
func createTestZip(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TestZip.zip")

	// create a temporary dir for files in zip archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer
	archive, _ := os.Create(targetFile)
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
func createTestNonArchive(dstDir string) string {
	targetFile := filepath.Join(dstDir, "test.txt")
	createTestFile(targetFile, "foo bar test")
	return targetFile
}

// createTestFile is a helper function to generate test files
func createTestFile(path string, content string) {
	byteArray := []byte(content)
	err := os.WriteFile(path, byteArray, 0644)
	if err != nil {
		panic(err)
	}
}

// createTestTar is a helper function to generate test data
func createTestTar(dstDir string) string {

	targetFile := filepath.Join(dstDir, "TarNormal.tar")

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer

	f, _ := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	tarWriter := tar.NewWriter(f)

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

	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// prepare generated zip+writer

	f, _ := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
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

func TestUnpack(t *testing.T) {

	// create test zip
	testDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		t.Errorf(err.Error())
	}
	testDir = filepath.Clean(testDir) + string(os.PathSeparator)
	defer os.RemoveAll(testDir)

	// create test zip
	f, _ := os.Open(createTestZip(testDir))
	defer f.Close()

	// perform actual tests
	err = Unpack(context.Background(), f, testDir, config.NewConfig())
	if err != nil {
		t.Errorf(err.Error())
	}
}

// TestUnpack is a test function
func TestUnpackOnTarget(t *testing.T) {

	// test cases
	cases := []struct {
		name        string
		fn          func(string) string
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
			testDir, err := os.MkdirTemp(os.TempDir(), "fooo*")
			if err != nil {
				panic(err)
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			// prepare vars
			want := tc.expectError

			// perform actual tests
			archive, err := os.Open(tc.fn(testDir))
			if err != nil {
				panic(err)
			}
			err = UnpackOnTarget(
				context.Background(),
				archive,
				testDir,
				target.NewOs(),
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

func TestGetHeader(t *testing.T) {
	tests := []struct {
		name    string
		src     io.Reader
		wantErr bool
	}{
		{
			name:    "Read header from bytes.Buffer (implements io.Seeker)",
			src:     bytes.NewBuffer([]byte("test data")),
			wantErr: false,
		},
		{
			name:    "Read header from bytes.Reader (implements io.Seeker)",
			src:     bytes.NewReader([]byte("test data")),
			wantErr: false,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := getHeader(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("getHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func gen1024ByteGzip(dstDir string) string {
	testFile := filepath.Join(dstDir, "GzipWithFile.gz")
	createGzip(testFile, strings.NewReader(strings.Repeat("A", 1024)))
	return testFile
}

func genSingleFileTar(dstDir string) string {
	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

	// create test file
	testFile := filepath.Join(tmpDir, "testFile")
	createTestFile(testFile, strings.Repeat("A", 1024))

	tarFileName := filepath.Join(dstDir, "TarNormalSingleFile.tar")
	createTestTarWithFiles(tarFileName, map[string]string{"TestFile": testFile})
	return tarFileName
}

func genTarGzWith5Files(dstDir string) string {
	// create a temporary dir for files in tar archive
	tmpDir := target.CreateTmpDir()
	defer os.RemoveAll(tmpDir)

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

// TestMetriksHook is a test function for the metriks hook
func TestMetriksHook(t *testing.T) {
	cases := []struct {
		name                  string
		inputGenerator        func(string) string
		inputName             string
		dst                   string
		WithContinueOnError   bool
		WithCreateDestination bool
		WithMaxExtractionSize int64
		WithMaxFiles          int64
		WithOverwrite         bool
		WithNoTarGzExtract    bool
		expectedMetrics       config.Metrics
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
			expectedMetrics: config.Metrics{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gzip",
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
			expectedMetrics: config.Metrics{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gzip",
			},
			expectError: false,
		},
		{
			name:                  "normal gzip with file and decompression target-name in sub-dir failing",
			inputGenerator:        gen1024ByteGzip,
			inputName:             "GzipWithFile.gz",
			dst:                   "sub/target", // important: the gzip decompression has a filename das dst
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: 1024,
			WithMaxFiles:          1,
			WithOverwrite:         false,
			expectedMetrics: config.Metrics{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 1,
				ExtractionSize:   0,
				ExtractedType:    "gzip",
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
			expectedMetrics: config.Metrics{
				ExtractedDirs:    0,
				ExtractedFiles:   1,
				ExtractionErrors: 0,
				ExtractionSize:   1024,
				ExtractedType:    "gzip",
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
			expectedMetrics: config.Metrics{
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
			expectedMetrics: config.Metrics{
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
			WithMaxExtractionSize: -1, // no limit, remark: the .tar > expectedMetrics.ExtractionSize
			WithMaxFiles:          5,
			WithOverwrite:         false,
			expectedMetrics: config.Metrics{
				ExtractedDirs:    0,
				ExtractedFiles:   5,
				ExtractionErrors: 0,
				ExtractionSize:   1024 * 5,
				ExtractedType:    "tar+gzip",
			},
			expectError: false,
		},
		{
			name:                  "normal tar.gz with file with max files limit",
			inputGenerator:        genTarGzWith5Files,
			dst:                   ".",
			WithContinueOnError:   false,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the .tar > expectedMetrics.ExtractionSize
			WithMaxFiles:          4,
			WithOverwrite:         false,
			expectedMetrics: config.Metrics{
				ExtractedDirs:    0,
				ExtractedFiles:   4,
				ExtractionErrors: 1,
				ExtractionSize:   1024 * 4,
				ExtractedType:    "tar+gzip",
			},
			expectError: true,
		},
		{
			name:                  "normal tar.gz with file failing bc/ of missing sub directory",
			inputGenerator:        genTarGzWith5Files,
			dst:                   "sub",
			WithContinueOnError:   true,
			WithCreateDestination: false,
			WithMaxExtractionSize: -1, // no limit, remark: the .tar > expectedMetrics.ExtractionSize
			WithMaxFiles:          5,
			WithOverwrite:         false,
			expectedMetrics: config.Metrics{
				ExtractedDirs:    0,
				ExtractedFiles:   0,
				ExtractionErrors: 5,
				ExtractionSize:   0,
				ExtractedType:    "tar+gzip",
			},
			expectError: false,
		},
		{
			name:                  "normal zip file",
			inputGenerator:        createTestZip,
			dst:                   ".",
			WithMaxFiles:          1,
			WithMaxExtractionSize: 14,
			expectedMetrics: config.Metrics{
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
			expectedMetrics: config.Metrics{
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
			testDir, err := os.MkdirTemp(os.TempDir(), "extraction_test_dir*")
			if err != nil {
				panic(err)
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			// open file
			archive, err := os.Open(tc.inputGenerator(testDir))
			if err != nil {
				panic(err)
			}

			// prepare config
			var collectedMetrics *config.Metrics
			hook := func(ctx context.Context, metrics *config.Metrics) {
				collectedMetrics = metrics
			}

			cfg := config.NewConfig(
				config.WithContinueOnError(tc.WithContinueOnError),
				config.WithCreateDestination(tc.WithCreateDestination),
				config.WithMaxExtractionSize(tc.WithMaxExtractionSize),
				config.WithMaxFiles(tc.WithMaxFiles),
				config.WithOverwrite(tc.WithOverwrite),
				config.WithNoTarGzExtract(tc.WithNoTarGzExtract),
				config.WithMetricsHook(hook),
			)

			// perform actual tests
			ctx := context.Background()
			dstDir := filepath.Join(testDir, tc.dst)
			err = UnpackOnTarget(ctx, archive, dstDir, target.NewOs(), cfg)

			// check if error is expected
			if tc.expectError != (err != nil) {
				t.Errorf("test case %d failed: %s\nexpected error: %v\ngot: %s", i, tc.name, tc.expectError, err)
			}

			// compare collected and expected metrics ExtractedFiles
			if collectedMetrics.ExtractedFiles != tc.expectedMetrics.ExtractedFiles {
				t.Errorf("test case %d failed: %s (ExtractedFiles)\nexpected: %v\ngot: %v", i, tc.name, tc.expectedMetrics.ExtractedFiles, collectedMetrics.ExtractedFiles)
			}

			// compare collected and expected metrics ExtractionErrors
			if collectedMetrics.ExtractionErrors != tc.expectedMetrics.ExtractionErrors {
				t.Errorf("test case %d failed: %s (ExtractionErrors)\nexpected: %v\ngot: %v", i, tc.name, tc.expectedMetrics.ExtractionErrors, collectedMetrics.ExtractionErrors)
			}

			// compare collected and expected metrics ExtractionSize
			if collectedMetrics.ExtractionSize != tc.expectedMetrics.ExtractionSize {
				t.Errorf("test case %d failed: %s (ExtractionSize [e:%v|g:%v])\nexpected: %v\ngot: %v", i, tc.name, tc.expectedMetrics.ExtractionSize, collectedMetrics.ExtractionSize, tc.expectedMetrics.ExtractionSize, collectedMetrics.ExtractionSize)
			}

			// compare collected and expected metrics ExtractedType
			if collectedMetrics.ExtractedType != tc.expectedMetrics.ExtractedType {
				t.Errorf("test case %d failed: %s (ExtractedType)\nexpected: %v\ngot: %v", i, tc.name, tc.expectedMetrics.ExtractedType, collectedMetrics.ExtractedType)
			}

		})
	}
}
