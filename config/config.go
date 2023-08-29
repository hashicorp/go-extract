package config

import (
	"fmt"
)

type ConfigOption func(*Config)

type Config struct {
	// MaxFiles is the maximum of files in an archive.
	// Set value to -1 to disable the check.
	MaxFiles int64

	// MaxExtractionSize is the maximum size of a file after decompression.
	// Set value to -1 to disable the check.
	MaxExtractionSize int64

	// Maximum time in seconds that an extraction should need to finish
	MaxExtractionTime int64

	// Define if files should be overwritten in the Destination
	Force bool
}

func NewConfig(opts ...ConfigOption) *Config {
	const (
		maxFiles          = 1000
		maxExtractionSize = 1 << (10 * 3) // 1 Gb
		maxExtractionTime = 60            // 1 minute
		force             = false
	)

	config := &Config{
		MaxFiles:          maxFiles,
		MaxExtractionSize: maxExtractionSize,
		MaxExtractionTime: maxExtractionTime,
		Force:             force,
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
func WithMaxExtractionSize(maxFileSize int64) ConfigOption {
	return func(c *Config) {
		c.MaxExtractionSize = maxFileSize
	}
}

func WithForce(enable bool) ConfigOption {
	return func(c *Config) {
		c.Force = enable
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
	if e.MaxExtractionSize == -1 {
		return nil
	}

	// check value
	if fileSize > e.MaxExtractionSize {
		return fmt.Errorf("maximum filesize exceeded")
	}
	return nil
}
