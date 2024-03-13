package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestMetricsString tests the String method of the Metrics struct
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

// TestSubmitMetrics tests the SubmitMetrics function
func TestSubmitMetrics(t *testing.T) {

	metricsReceived := false

	// Create a new Metrics instance
	m := &Metrics{ExtractedType: "tar"}

	// Add a processor to the metrics
	m.AddProcessor(func(ctx context.Context, m *Metrics) {
		m.ExtractedType = fmt.Sprintf("%s.%s", m.ExtractedType, "gz")
	})

	// Call SubmitMetrics
	ApplyProcessorAndSubmit(context.Background(), m, func(ctx context.Context, m *Metrics) {
		metricsReceived = true
	})

	// Check if metrics were received
	if !metricsReceived {
		t.Error("Expected metrics to be received")
	}

	// Check if the processor was called
	if m.ExtractedType != "tar.gz" {
		t.Errorf("Expected 'tar.gz', but got '%s'", m.ExtractedType)
	}

}
