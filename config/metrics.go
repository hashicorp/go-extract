package config

import (
	"context"
	"encoding/json"
	"time"
)

// Metrics is a struct type that holds all metrics of an extraction
type Metrics struct {

	// ExtractedDirs is the number of extracted directories
	ExtractedDirs int64

	// ExtractionDuration is the time it took to extract the archive
	ExtractionDuration time.Duration

	// ExtractionErrors is the number of errors during extraction
	ExtractionErrors int64

	// ExtractedFiles is the number of extracted files
	ExtractedFiles int64

	// ExtractionSize is the size of the extracted files
	ExtractionSize int64

	// ExtractedSymlinks is the number of extracted symlinks
	ExtractedSymlinks int64

	// ExtractedType is the type of the archive
	ExtractedType string

	// InputSize is the size of the input
	InputSize int64

	// LastExtractionError is the last error during extraction
	LastExtractionError error

	// metricsProcessor performs operations on metrics before submitting to hook
	metricsProcessor []MetricsHook

	// PatternMismatches is the number of skipped files
	PatternMismatches int64

	// UnsupportedFiles is the number of skipped unsupported files
	UnsupportedFiles int64

	// LastUnsupportedFile is the last skipped unsupported file
	LastUnsupportedFile string
}

// String returns a string representation of the metrics
func (m Metrics) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

// MarshalJSON implements the json.Marshaler interface
func (m Metrics) MarshalJSON() ([]byte, error) {
	var lastError string
	if m.LastExtractionError != nil {
		lastError = m.LastExtractionError.Error()
	}

	type Alias Metrics
	return json.Marshal(&struct {
		LastExtractionError string `json:"LastExtractionError"`
		*Alias
	}{
		LastExtractionError: lastError,
		Alias:               (*Alias)(&m),
	})
}

// MetricsHook emits metrics to hook and applies all registered metricsProcessor
func (m *Metrics) Submit(ctx context.Context, hook MetricsHook) {

	// process metrics in reverse order
	for i := len(m.metricsProcessor) - 1; i >= 0; i-- {
		m.metricsProcessor[i](ctx, m)
	}

	// emit metrics
	if hook != nil {
		hook(ctx, m)
	}
}

// AddMetricsProcessor adds a metrics processor to the config
func (m *Metrics) AddProcessor(hook MetricsHook) {
	m.metricsProcessor = append(m.metricsProcessor, hook)
}
