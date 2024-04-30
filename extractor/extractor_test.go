package extractor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

func TestMatchesMagicBytes(t *testing.T) {
	cases := []struct {
		name        string
		data        []byte
		magicBytes  [][]byte
		offset      int
		expectMatch bool
	}{
		{
			name:        "match",
			data:        []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			magicBytes:  [][]byte{{0x02, 0x03}},
			offset:      2,
			expectMatch: true,
		},
		{
			name:        "missmatch",
			data:        []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			magicBytes:  [][]byte{{0x02, 0x03}},
			offset:      1,
			expectMatch: false,
		},
		{
			name:        "to few data to match",
			data:        []byte{0x00},
			magicBytes:  [][]byte{{0x02, 0x03}},
			offset:      1,
			expectMatch: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			expected := tc.expectMatch
			got := matchesMagicBytes(tc.data, tc.offset, tc.magicBytes)

			// success if both are nil and no engine found
			if got != expected {
				t.Errorf("test case %d failed: %s!", i, tc.name)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	c := config.NewConfig(config.WithContinueOnError(false))
	td := &telemetry.Data{}

	err := errors.New("test error")
	if err := handleError(c, td, "test message", err); err == nil {
		t.Error("handleError should return an error when continueOnError is false")
	}

	if td.ExtractionErrors != int64(1) {
		t.Error("ExtractionErrors was not incremented")
	}

	if td.LastExtractionError.Error() != "test message: test error" {
		t.Error("LastExtractionError was not set correctly")
	}

	c = config.NewConfig(config.WithContinueOnError(true))
	err = handleError(c, td, "test message", err)
	if err != nil {
		t.Error("handleError should return nil when continueOnError is true")
	}
}

func TestCheckPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "No patterns given",
			patterns: []string{},
			path:     "test/path",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "Path matches pattern",
			patterns: []string{"test/*"},
			path:     "test/path",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "Path does not match pattern",
			patterns: []string{"other/*"},
			path:     "test/path",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Invalid pattern",
			patterns: []string{"["},
			path:     "test/path",
			want:     false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPatterns(tt.patterns, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPatterns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPatterns() = %v, want %v", got, tt.want)
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

// createByteReader creates a reader for the given data
func createByteReader(target string, data []byte) io.Reader {
	return bytes.NewReader(data)
}

type simpleReader struct {
	r io.Reader
}

func (s *simpleReader) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}

// createByteReader creates a reader for the given data
func createSimpleReader(target string, data []byte) io.Reader {
	return &simpleReader{r: createByteReader(target, data)}
}

func TestValidFilename(t *testing.T) {

	// prepare test content
	testFileNames := []string{
		"CON", "PRN", "AUX", "NUL", "LPT", "COM",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		".", "..",
	}
	testIvalidChraracters := []string{
		`<`, `>`, `:`, `"`, `|`, `?`, `*`, `/`, `\`, ` `, `.`,
	}
	nameBase := "test"
	// add invalid characters to the test file names end
	for _, invalidChar := range testIvalidChraracters {
		testFileNames = append(testFileNames, nameBase+invalidChar)
	}
	// add invalid characters to the test file names start
	for _, invalidChar := range testIvalidChraracters {
		testFileNames = append(testFileNames, invalidChar+nameBase)
	}

	// run tests
	for i, name := range testFileNames {

		// create a file with the given name
		invalid := false
		tmpDir := t.TempDir()
		testFilePath := tmpDir + string(os.PathSeparator) + name

		// try to create a file with the given name
		testFile, err := os.Create(testFilePath)
		if err != nil {
			invalid = true
		}
		if err == nil {
			// If the directory is a character device (like the printer port), treat it as an error
			info, statError := os.Stat(testFilePath)
			if statError == nil && info.Mode()&fs.ModeCharDevice != 0 {
				err = fmt.Errorf("file is a character device")
				invalid = true
			}
		}
		defer func() {
			if testFile != nil {
				if err := testFile.Close(); err != nil {
					t.Errorf("error closing file: %v", err)
				}
			}
		}()

		// evaluate test case``
		if invalid != !validFilename(name) {
			t.Errorf("test case %d failed: err=%v and validFilename(%s): %t", i, err, name, validFilename(name))
		}
	}

}
