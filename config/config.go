package config

import (
	"fmt"
)

// ConfigOption is a function pointer to implement the option pattern
type ConfigOption func(*Config)

// Config is a struct type that holds all config options
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

	// DenySymlinks offers the option to disable the extraction of symlinks
	DenySymlinks bool

	// ContinueOnError decides if the extraction should be continued even if an error occoured
	ContinueOnError bool
}

// NewConfig is a generator option that takes opts as adjustments of the
// default configuration in an option pattern style
func NewConfig(opts ...ConfigOption) *Config {
	const (
		maxFiles          = 1000
		maxExtractionSize = 1 << (10 * 3) // 1 Gb
		maxExtractionTime = 60            // 1 minute
		force             = false
		denySymlinks      = false
		continueOnError   = false
	)

	config := &Config{
		MaxFiles:          maxFiles,
		MaxExtractionSize: maxExtractionSize,
		MaxExtractionTime: maxExtractionTime,
		Force:             force,
		DenySymlinks:      denySymlinks,
		ContinueOnError:   continueOnError,
	}

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}

	// return the modified house instance
	return config
}

// WithMaxFiles options pattern function to set maxFiles in the config
func WithMaxFiles(maxFiles int64) ConfigOption {
	return func(c *Config) {
		c.MaxFiles = maxFiles
	}
}

// WithMaxExtractionTime options pattern function to set WithMaxExtractionTime in the config
func WithMaxExtractionTime(maxExtractionTime int64) ConfigOption {
	return func(c *Config) {
		c.MaxExtractionTime = maxExtractionTime
	}
}

// WithMaxExtractionSize options pattern function to set WithMaxExtractionSize in the config
func WithMaxExtractionSize(maxExtractionSize int64) ConfigOption {
	return func(c *Config) {
		c.MaxExtractionSize = maxExtractionSize
	}
}

// WithForce options pattern function to set force in the config
func WithForce(enable bool) ConfigOption {
	return func(c *Config) {
		c.Force = enable
	}
}

// WithDenySymlinks options pattern function to deny symlink extraction
func WithDenySymlinks(deny bool) ConfigOption {
	return func(c *Config) {
		c.DenySymlinks = deny
	}
}

// WithContinueOnError options pattern function to continue on error during extraction
func WithContinueOnError(yes bool) ConfigOption {
	return func(c *Config) {
		c.ContinueOnError = yes
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
func (e *Config) CheckExtractionSize(fileSize int64) error {

	// check if disabled
	if e.MaxExtractionSize == -1 {
		return nil
	}

	// check value
	if fileSize > e.MaxExtractionSize {
		return fmt.Errorf("maximum extraction size exceeded")
	}
	return nil
}
