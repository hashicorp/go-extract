package extract_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-extract"
)

// TestDataString tests the String method of the data struct
func TestDataString(t *testing.T) {
	m := extract.TelemetryData{
		ExtractedType:       "tar",
		ExtractionDuration:  time.Duration(5 * time.Millisecond),
		ExtractionSize:      1024,
		ExtractedFiles:      5,
		ExtractedSymlinks:   2,
		ExtractedDirs:       1,
		ExtractionErrors:    1,
		LastExtractionError: fmt.Errorf("example error"),
		InputSize:           2048,
		UnsupportedFiles:    0,
	}

	expected := `{"LastExtractionError":"example error","ExtractedDirs":1,"ExtractionDuration":5000000,"ExtractionErrors":1,"ExtractedFiles":5,"ExtractionSize":1024,"ExtractedSymlinks":2,"ExtractedType":"tar","InputSize":2048,"PatternMismatches":0,"UnsupportedFiles":0,"LastUnsupportedFile":""}`
	if m.String() != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, m.String())
	}
}
