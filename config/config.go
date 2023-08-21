package config

import (
	"fmt"
)

type ConfigOption func(*Config)

type Config struct {
	// MaxFiles is the maximum of files in an archive.
	// Set value to -1 to disable the check.
	MaxFiles int64

	// MaxFileSize is the maximum size of a file after decompression.
	// Set value to -1 to disable the check.
	MaxFileSize int64

	// Maximum time in seconds that an extraction should need to finish
	MaxExtractionTime int64

	// Define if files should be overwritten in the Destination
	Overwrite bool
}

func NewConfig(opts ...ConfigOption) *Config {
	const (
		maxFiles          = 1000
		maxFileSize       = 1 << (10 * 3) // 1 Gb
		maxExtractionTime = 60            // 1 minute
	)

	config := &Config{
		MaxFiles:          maxFiles,
		MaxFileSize:       maxFileSize,
		MaxExtractionTime: maxExtractionTime,
	}

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}

	// return the modified house instance
	return config
}

func WithMaxFiles(maxFiles int64) ConfigOption {
	return func(c *Config) {
		c.MaxFiles = maxFiles
	}
}

func WithMaxExtractionTime(maxExtractionTime int64) ConfigOption {
	return func(c *Config) {
		c.MaxExtractionTime = maxExtractionTime
	}
}
func WithMaxFileSize(maxFileSize int64) ConfigOption {
	return func(c *Config) {
		c.MaxFileSize = maxFileSize
	}
}

func WithOverwrite() ConfigOption {
	return func(c *Config) {
		c.Overwrite = true
	}
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
