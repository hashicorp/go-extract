package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"
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

	// Log stream for extraction
	Log *slog.Logger

	// LogLevel is the log level for the logger
	LogLevel slog.LevelVar
}

// NewConfig is a generator option that takes opts as adjustments of the
// default configuration in an option pattern style
func NewConfig(opts ...ConfigOption) *Config {
	const (
		continueOnError   = false
		allowSymlinks     = true
		followSymlinks    = false
		logLevel          = slog.LevelInfo
		maxFiles          = 1000          // 1k files
		maxExtractionSize = 1 << (10 * 3) // 1 Gb
		maxExtractionTime = 60            // 1 minute
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
		Verbose:           verbose,
	}

	// setup default logger
	config.LogLevel.Set(logLevel)
	logOpts := &slog.HandlerOptions{
		Level: &config.LogLevel,
	}
	logHandler := slog.NewTextHandler(os.Stdout, logOpts)
	config.Log = slog.New(logHandler)
	// config.Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	// 	Level: &config.LogLevel,
	// }))

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}

	// return the modified house instance
	return config
}

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

// MetricsHook is a function pointer to implement the option pattern
type MetricsHook func(Metrics)

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

// WithLogLevel options pattern function to get details on extraction
func WithLogLevel(logLevel slog.Level) ConfigOption {
	return func(c *Config) {
		c.LogLevel.Set(logLevel)
	}
}

// WithLogger options pattern function to set a custom logger
func WithLogger(logger *slog.Logger) ConfigOption {
	return func(c *Config) {
		c.Log = logger
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
