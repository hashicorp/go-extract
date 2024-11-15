// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"encoding/json"
	"time"
)

// TelemetryData holds all telemetry data of an extraction.
type TelemetryData struct {
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

// String returns a string representation of [TelemetryData].
func (m TelemetryData) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

// MarshalJSON implements the [encoding/json.Marshaler] interface.
func (m TelemetryData) MarshalJSON() ([]byte, error) {
	var lastError string
	if m.LastExtractionError != nil {
		lastError = m.LastExtractionError.Error()
	}

	type Alias TelemetryData
	return json.Marshal(&struct {
		LastExtractionError string `json:"LastExtractionError"`
		*Alias
	}{
		LastExtractionError: lastError,
		Alias:               (*Alias)(&m),
	})
}

// TelemetryHook is a function type that performs operations on [TelemetryData]
// after an extraction has finished which can be used to submit the [TelemetryData]
// to a telemetry service, for example.
type TelemetryHook func(context.Context, *TelemetryData)

// Equals returns true if the given [TelemetryData] is equal to the receiver.
func (td *TelemetryData) Equals(other *TelemetryData) bool {
	if td == nil && other == nil {
		return true
	}
	if td == nil || other == nil {
		return false
	}
	return td.ExtractedDirs == other.ExtractedDirs &&
		td.ExtractionErrors == other.ExtractionErrors &&
		td.ExtractedFiles == other.ExtractedFiles &&
		td.ExtractionSize == other.ExtractionSize &&
		td.ExtractedSymlinks == other.ExtractedSymlinks &&
		td.ExtractedType == other.ExtractedType &&
		td.PatternMismatches == other.PatternMismatches &&
		td.UnsupportedFiles == other.UnsupportedFiles &&
		td.LastUnsupportedFile == other.LastUnsupportedFile
}
