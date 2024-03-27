package extractor

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

// TestIs7zip tests the Is7zip function
func TestIs7zip(t *testing.T) {

	// test cases
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00}, false},
	}

	// run tests
	for _, tt := range tests {
		if got := Is7zip(tt.header); got != tt.want {
			t.Errorf("Is7zip(%v) = %v; want %v", tt.header, got, tt.want)
		}
	}

}

// TestUnpack7zip tests the Unpack7zip function
func TestUnpack7zip(t *testing.T) {

	archiveName := "test.7z"
	archiveBytes, err := hex.DecodeString("377abcaf271c00049af18e7973000000000000002000000000000000a7e80f9801000b48656c6c6f20576f726c6421000000813307ae0fcef2b20c07c8437f41b1fafddb88b6d7636b8bd58a0e24a2f717a5f156e37f41fd00833298421d5d088c0cf987b30c0473663599e4d2f21cb69620038f10458109662135c3024189f42799abe3227b174a853e824f808b2efaab000017061001096300070b01000123030101055d001000000c760a015bcfa0a70000")
	archivedFile := "test/data"
	archivedFileContent := "Hello World!"

	if err != nil {
		t.Fatal(err)
	}

	tc := []struct {
		name        string
		generator   func(target string, data []byte) io.Reader
		content     []byte
		c           *config.Config
		expectError bool
	}{
		{
			name:        "unpack 7zip",
			generator:   createFile,
			expectError: false,
		},
		{
			name:        "unpack 7zip from memory",
			generator:   createByteReader,
			expectError: false,
		},
		{
			name:        "unpack 7zip caching needed (file)",
			generator:   createSimpleReader,
			expectError: false,
		},
		{
			name:        "unpack 7zip caching needed (memory)",
			generator:   createSimpleReader,
			c:           config.NewConfig(config.WithCacheInMemory(true)),
			expectError: false,
		},
		{
			name:        "unpack 7zip, input size limit",
			generator:   createByteReader,
			c:           config.NewConfig(config.WithMaxInputSize(25)),
			expectError: true,
		},
		{
			name:        "unpack 7zip, output size limit",
			generator:   createByteReader,
			c:           config.NewConfig(config.WithMaxExtractionSize(5)),
			expectError: true,
		},
		{
			name:        "unpack 7zip, invalid archive",
			generator:   createByteReader,
			content:     []byte("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {

			// check content
			if len(tt.content) == 0 {
				tt.content = archiveBytes
			} else {
				fmt.Println(tt.content)
			}

			// create a temporary directory with archive
			tmpDir := t.TempDir()
			r := tt.generator(filepath.Join(tmpDir, archiveName), tt.content)
			defer func() {
				if c, ok := r.(io.Closer); ok {
					c.Close()
				}
			}()

			// check config
			if tt.c == nil {
				tt.c = config.NewConfig()
			}

			// unpack archive
			err := Unpack7Zip(context.Background(), r, tmpDir, tt.c)
			if tt.expectError != (err != nil) {
				t.Errorf("%v: expected error: %v, got: %v", tt.name, tt.expectError, err)
			}

			if !tt.expectError {
				// check if file was extracted
				content, err := os.ReadFile(filepath.Join(tmpDir, archivedFile))
				if err != nil {
					t.Fatal(err)
				}
				if string(content) != archivedFileContent {
					t.Errorf("expected content: '%v', got: '%v'", archivedFileContent, string(content))
				}
			}
		})
	}

}
