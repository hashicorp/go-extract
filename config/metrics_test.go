package config

import (
	"testing"
	"time"
)

func TestMetricsString(t *testing.T) {
	m := Metrics{
		ExtractedType:       "tar",
		ExtractionDuration:  time.Duration(5 * time.Millisecond),
		ExtractionSize:      1024,
		ExtractedFiles:      5,
		ExtractedSymlinks:   2,
		ExtractedDirs:       1,
		ExtractionErrors:    0,
		LastExtractionError: nil,
		InputSize:           2048,
	}

	expected := `{"ExtractedDirs":1,"ExtractionDuration":5000000,"ExtractionErrors":0,"ExtractedFiles":5,"ExtractionSize":1024,"ExtractedSymlinks":2,"ExtractedType":"tar","InputSize":2048,"LastExtractionError":null}`
	if m.String() != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, m.String())
	}
}