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
	// allowSymlinks offers the option to enable/disable the extraction of symlinks
	allowSymlinks bool

	// continueOnError decides if the extraction should be continued even if an error occurred
	continueOnError bool

	// create destination directory if it does not exist
	createDestination bool

	// followSymlinks follow symlinks to directories during extraction
	followSymlinks bool

	// logger stream for extraction
	logger Logger

	// maxExtractionSize is the maximum size of a file after decompression.
	// Set value to -1 to disable the check.
	maxExtractionSize int64

	// maxFiles is the maximum of files in an archive.
	// Set value to -1 to disable the check.
	maxFiles int64

	// maxInputSize is the maximum size of the input
	// Set value to -1 to disable the check.
	maxInputSize int64

	// MetricsHook is a function pointer to consume metrics after finished extraction
	// Important: do not adjust this value after extraction started
	metricsHooks []MetricsHook

	// Define if files should be overwritten in the destination
	overwrite bool

	// tarGzExtract offers the option to enable/disable the extraction of tar.gz archives
	tarGzExtract bool

	// verbose log extraction to stderr
	verbose bool
}

// NewConfig is a generator option that takes opts as adjustments of the
// default configuration in an option pattern style
func NewConfig(opts ...ConfigOption) *Config {
	const (
		allowSymlinks     = true
		continueOnError   = false
		createDestination = false
		followSymlinks    = false
		maxFiles          = 1000          // 1k files
		maxExtractionSize = 1 << (10 * 3) // 1 Gb
		maxExtractionTime = 60            // 1 minute
		maxInputSize      = 1 << (10 * 3) // 1 Gb
		overwrite         = false
		tarGzExtract      = false
		verbose           = false
	)

	// disable logging by default
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	// setup default values
	config := &Config{
		allowSymlinks:     allowSymlinks,
		continueOnError:   continueOnError,
		createDestination: createDestination,
		followSymlinks:    followSymlinks,
		logger:            logger,
		maxFiles:          maxFiles,
		maxExtractionSize: maxExtractionSize,
		maxInputSize:      maxInputSize,
		overwrite:         overwrite,
		tarGzExtract:      tarGzExtract,
		verbose:           verbose,
	}

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}

	// return the modified house instance
	return config
}

// MetricsHook is a function pointer to implement the option pattern
type MetricsHook func(context.Context, *Metrics)

// WithMetricsHook options pattern function to set a metrics hook
func WithMetricsHook(hook MetricsHook) ConfigOption {
	return func(c *Config) {
		c.metricsHooks = append(c.metricsHooks, hook)
	}
}

// WithMaxFiles options pattern function to set maxFiles in the config (-1 to disable check)
func WithMaxFiles(maxFiles int64) ConfigOption {
	return func(c *Config) {
		c.maxFiles = maxFiles
	}
}

// WithTarGzExtract options pattern function to enable/disable tar.gz extraction
func WithTarGzExtract(enable bool) ConfigOption {
	return func(c *Config) {
		c.tarGzExtract = enable
	}
}

// AllowSymlinks returns true if symlinks are allowed
func (c *Config) AllowSymlinks() bool {
	return c.allowSymlinks
}

// ContinueOnError returns true if the extraction should continue on error
func (c *Config) ContinueOnError() bool {
	return c.continueOnError
}

// CreateDestination returns true if the destination directory should be created if it does not exist
func (c *Config) CreateDestination() bool {
	return c.createDestination
}

// FollowSymlinks returns true if symlinks should be followed
func (c *Config) FollowSymlinks() bool {
	return c.followSymlinks
}

func (c *Config) Logger() Logger {
	return c.logger
}

// MaxExtractionSize returns the maximum size of a file after decompression
func (c *Config) MaxExtractionSize() int64 {
	return c.maxExtractionSize
}

// MaxFiles returns the maximum of files in an archive
func (c *Config) MaxFiles() int64 {
	return c.maxFiles
}

// MaxInputSize returns the maximum size of the input
func (c *Config) MaxInputSize() int64 {
	return c.maxInputSize
}

// Overwrite returns true if files should be overwritten in the destination
func (c *Config) Overwrite() bool {
	return c.overwrite
}

// TarGzExtract returns true if tar.gz extraction is enabled
func (c *Config) TarGzExtract() bool {
	return c.tarGzExtract
}

// MetricsHooksOnce emits metrics once to all configured hooks and resets the hook list
// after execution
func (c *Config) MetricsHooksOnce(ctx context.Context, metrics *Metrics) {

	// emit metrics in reverse order
	for i := len(c.metricsHooks) - 1; i >= 0; i-- {
		c.metricsHooks[i](ctx, metrics)
	}

	// empty hooks
	c.metricsHooks = []MetricsHook{}

}

func (c *Config) AddMetricsHook(hook MetricsHook) {
	c.metricsHooks = append(c.metricsHooks, hook)
}

// WithMaxExtractionSize options pattern function to set WithMaxExtractionSize in the
// config (-1 to disable check)
func WithMaxExtractionSize(maxExtractionSize int64) ConfigOption {
	return func(c *Config) {
		c.maxExtractionSize = maxExtractionSize
	}
}

// WithMaxInputSize options pattern function to set MaxInputSize in the
// config (-1 to disable check)
func WithMaxInputSize(maxInputSize int64) ConfigOption {
	return func(c *Config) {
		c.maxInputSize = maxInputSize
	}
}

// WithOverwrite options pattern function to set overwrite in the config
func WithOverwrite(enable bool) ConfigOption {
	return func(c *Config) {
		c.overwrite = enable
	}
}

// WithAllowSymlinks options pattern function to deny symlink extraction
func WithAllowSymlinks(allow bool) ConfigOption {
	return func(c *Config) {
		c.allowSymlinks = allow
	}
}

// WithContinueOnError options pattern function to continue on error during extraction
func WithContinueOnError(yes bool) ConfigOption {
	return func(c *Config) {
		c.continueOnError = yes
	}
}

// WithFollowSymlinks options pattern function to follow symlinks to  directories during extraction
func WithFollowSymlinks(follow bool) ConfigOption {
	return func(c *Config) {
		c.followSymlinks = follow
	}
}

// WithLogger options pattern function to set a custom logger
func WithLogger(logger Logger) ConfigOption {
	return func(c *Config) {
		c.logger = logger
	}
}

// checkMaxFiles checks if counter exceeds the MaxFiles of the Extractor e
func (e *Config) CheckMaxObjects(counter int64) error {

	// check if disabled
	if e.MaxFiles() == -1 {
		return nil
	}

	// check value
	if counter > e.MaxFiles() {
		return fmt.Errorf("to many files in archive")
	}
	return nil
}

// checkFileSize checks if fileSize exceeds the MaxFileSize of the Extractor e
func (e *Config) CheckExtractionSize(fileSize int64) error {

	// check if disabled
	if e.MaxExtractionSize() == -1 {
		return nil
	}

	// check value
	if fileSize > e.MaxExtractionSize() {
		return fmt.Errorf("maximum extraction size exceeded")
	}
	return nil
}

// WithCreateDestination options pattern function to create destination directory if it does not exist
func WithCreateDestination(create bool) ConfigOption {
	return func(c *Config) {
		c.createDestination = create
	}
}
