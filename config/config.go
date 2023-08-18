package config

import (
	"fmt"
)

type Config struct {
	// MaxFiles is the maximum of files in an archive.
	// Set value to -1 to disable the check.
	MaxFiles int64

	// MaxFileSize is the maximum size of a file after decompression.
	// Set value to -1 to disable the check.
	MaxFileSize int64

	// Maximum time in seconds that an extraction should need to finish
	MaxExtractionTime int64
}

// Default creates a new Extract object with defaults
func Default() *Config {
	return &Config{
		MaxFiles:          1000,
		MaxFileSize:       1 << (10 * 3), // 1 Gb
		MaxExtractionTime: 60,            // 1 minute
	}
}

// Default creates a new Extract object with defaults
func SafeDefault() *Config {
	return Default()
}

// checkMaxFiles checks if counter exceeds the MaxFiles of the Extractor e
func (e *Config) CheckMaxFiles(counter int64) error {

	// check if disabled
	if e.MaxFiles == -1 {
		return nil
	}

	// check value
	if counter > e.MaxFiles {
		return fmt.Errorf("to many files in archive")
	}
	return nil
}

// checkFileSize checks if fileSize exceeds the MaxFileSize of the Extractor e
func (e *Config) CheckFileSize(fileSize int64) error {

	// check if disabled
	if e.MaxFileSize == -1 {
		return nil
	}

	// check value
	if fileSize > e.MaxFileSize {
		return fmt.Errorf("maximum filesize exceeded")
	}
	return nil
}
