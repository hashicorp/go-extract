package metrics

import (
	"context"
	"fmt"
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

func TestSubmit(t *testing.T) {

	ctx := context.Background()
	processorExecuted := false

	// Create a new Metrics instance
	m := Metrics{ExtractedType: "tar"}

	// Add a processor
	m.AddProcessor(func(ctx context.Context, m *Metrics) {
		processorExecuted = true
	})

	// Apply hooks
	m.ApplyProcessor(ctx)

	// Check if processor was executed
	if !processorExecuted {
		t.Error("Expected processor to be executed")
	}
}
