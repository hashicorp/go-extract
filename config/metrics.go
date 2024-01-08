package config

import (
	"fmt"
	"time"
)

// Metrics is a struct type that holds all metrics of an extraction
type Metrics struct {

	// ExtractionDuration is the time it took to extract the archive
	ExtractionDuration time.Duration

	// ExtractionSize is the size of the extracted files
	ExtractionSize int64

	// ExtractedType is the type of the archive
	ExtractedType string

	// ExtractedFiles is the number of extracted files
	ExtractedFiles int64

	// ExtractedSymlinks is the number of extracted symlinks
	ExtractedSymlinks int64

	// ExtractedDirs is the number of extracted directories
	ExtractedDirs int64

	// ExtractionErrors is the number of errors during extraction
	ExtractionErrors int64

	// LastExtractionError is the last error during extraction
	LastExtractionError error
}

// String returns a string representation of the metrics
func (m Metrics) String() string {
	return fmt.Sprintf("type: %s, duration: %s, size: %d, files: %d, symlinks: %d, dirs: %d, errors: %d, last error: %s",
		m.ExtractedType,
		m.ExtractionDuration,
		m.ExtractionSize,
		m.ExtractedFiles,
		m.ExtractedSymlinks,
		m.ExtractedDirs,
		m.ExtractionErrors,
		m.LastExtractionError,
	)
}
