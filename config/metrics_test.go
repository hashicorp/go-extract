package config

import (
	"testing"
	"time"
)

func TestMetricsString(t *testing.T) {
	m := Metrics{
		ExtractedType:       "tar",
		ExtractionDuration:  time.Duration(5 * time.Second),
		ExtractionSize:      1024,
		ExtractedFiles:      5,
		ExtractedSymlinks:   2,
		ExtractedDirs:       1,
		ExtractionErrors:    0,
		LastExtractionError: nil,
		InputSize:           2048,
	}

	expected := "type: tar, duration: 5s, size: 1024, files: 5, symlinks: 2, dirs: 1, errors: 0, last error: <nil>, input size: 2048"
	if m.String() != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, m.String())
	}
}
