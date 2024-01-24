package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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

	// MetricsHook is a function pointer to consume metrics after finished extraction
	MetricsHook MetricsHook

	// Define if files should be overwritten in the destination
	Overwrite bool

	// AllowSymlinks offers the option to enable/disable the extraction of symlinks
	AllowSymlinks bool

	// ContinueOnError decides if the extraction should be continued even if an error occurred
	ContinueOnError bool

	// FollowSymlinks follow symlinks to directories during extraction
	FollowSymlinks bool

	// Verbose log extraction to stderr
	Verbose bool

	// Logger stream for extraction
	Logger Logger

	// Create destination directory if it does not exist
	CreateDestination bool

	// MaxInputFileSize is the maximum size of the input file
	// Set value to -1 to disable the check.
	MaxInputFileSize int64
}

// NewConfig is a generator option that takes opts as adjustments of the
// default configuration in an option pattern style
func NewConfig(opts ...ConfigOption) *Config {
	const (
		continueOnError   = false
		allowSymlinks     = true
		followSymlinks    = false
		maxFiles          = 1000          // 1k files
		maxExtractionSize = 1 << (10 * 3) // 1 Gb
		maxExtractionTime = 60            // 1 minute
		maxInputFileSize  = 1 << (10 * 3) // 1 Gb
		overwrite         = false
		verbose           = false
	)

	// setup default values
	config := &Config{
		ContinueOnError:   continueOnError,
		AllowSymlinks:     allowSymlinks,
		FollowSymlinks:    followSymlinks,
		Overwrite:         overwrite,
		MaxFiles:          maxFiles,
		MaxExtractionSize: maxExtractionSize,
		MaxInputFileSize:  maxInputFileSize,
		Verbose:           verbose,
	}

	// disable logging by default
	config.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}

	// return the modified house instance
	return config
}

// MetricsHook is a function pointer to implement the option pattern
type MetricsHook func(context.Context, Metrics)

// WithMetricsHook options pattern function to set a metrics hook
func WithMetricsHook(hook MetricsHook) ConfigOption {
	return func(c *Config) {
		c.MetricsHook = hook
	}
}

// WithMaxFiles options pattern function to set maxFiles in the config (-1 to disable check)
func WithMaxFiles(maxFiles int64) ConfigOption {
	return func(c *Config) {
		c.MaxFiles = maxFiles
	}
}

// WithMaxExtractionSize options pattern function to set WithMaxExtractionSize in the
// config (-1 to disable check)
func WithMaxExtractionSize(maxExtractionSize int64) ConfigOption {
	return func(c *Config) {
		c.MaxExtractionSize = maxExtractionSize
	}
}

// WithMaxInputFileSize options pattern function to set WithMaxInputFileSize in the
// config (-1 to disable check)
func WithMaxInputFileSize(maxInputFileSize int64) ConfigOption {
	return func(c *Config) {
		c.MaxInputFileSize = maxInputFileSize
	}
}

// WithOverwrite options pattern function to set overwrite in the config
func WithOverwrite(enable bool) ConfigOption {
	return func(c *Config) {
		c.Overwrite = enable
	}
}

// WithAllowSymlinks options pattern function to deny symlink extraction
func WithAllowSymlinks(allow bool) ConfigOption {
	return func(c *Config) {
		c.AllowSymlinks = allow
	}
}

// WithContinueOnError options pattern function to continue on error during extraction
func WithContinueOnError(yes bool) ConfigOption {
	return func(c *Config) {
		c.ContinueOnError = yes
	}
}

// WithFollowSymlinks options pattern function to follow symlinks to  directories during extraction
func WithFollowSymlinks(follow bool) ConfigOption {
	return func(c *Config) {
		c.FollowSymlinks = follow
	}
}

// WithLogger options pattern function to set a custom logger
func WithLogger(logger Logger) ConfigOption {
	return func(c *Config) {
		c.Logger = logger
	}
}

// checkMaxFiles checks if counter exceeds the MaxFiles of the Extractor e
func (e *Config) CheckMaxObjects(counter int64) error {

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

// WithCreateDestination options pattern function to create destination directory if it does not exist
func WithCreateDestination(create bool) ConfigOption {
	return func(c *Config) {
		c.CreateDestination = create
	}
}
