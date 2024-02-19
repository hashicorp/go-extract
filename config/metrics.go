package config

import (
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
		ExtractionDuration  int64  `json:"ExtractionDuration"`
		LastExtractionError string `json:"LastExtractionError"`
		*Alias
	}{
		ExtractionDuration:  m.ExtractionDuration.Microseconds(),
		LastExtractionError: lastError,
		Alias:               (*Alias)(&m),
	})
}
