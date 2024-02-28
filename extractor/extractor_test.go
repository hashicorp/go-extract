package extractor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/hashicorp/go-extract/config"
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
	metrics := &config.Metrics{}

	err := errors.New("test error")
	if err := handleError(c, metrics, "test message", err); err == nil {
		t.Error("handleError should return an error when continueOnError is false")
	}

	if metrics.ExtractionErrors != int64(1) {
		t.Error("ExtractionErrors was not incremented")
	}

	if metrics.LastExtractionError.Error() != "test message: test error" {
		t.Error("LastExtractionError was not set correctly")
	}

	c = config.NewConfig(config.WithContinueOnError(true))
	err = handleError(c, metrics, "test message", err)
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
func createFile(ctx context.Context, target string, data []byte) io.Reader {

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
func createByteReader(ctx context.Context, target string, data []byte) io.Reader {
	return bytes.NewReader(data)
}
