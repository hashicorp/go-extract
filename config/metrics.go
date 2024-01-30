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
}

// String returns a string representation of the metrics
func (m Metrics) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}
