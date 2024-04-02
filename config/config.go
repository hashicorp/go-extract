package config

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"

	"github.com/hashicorp/go-extract/telemetry"
)

// ConfigOption is a function pointer to implement the option pattern
type ConfigOption func(*Config)

// Config is a struct type that holds all config options
type Config struct {
	// cacheInMemory offers the option to enable/disable caching in memory. This applies only
	// to the extraction of zip archives, which are provided as a stream.
	cacheInMemory bool

	// continueOnError decides if the extraction should be continued even if an error occurred
	continueOnError bool

	// continueOnUnsupportedFiles offers the option to enable/disable skipping unsupported files
	continueOnUnsupportedFiles bool

	// create destination directory if it does not exist
	createDestination bool

	// defaultDirPermission is the default folder permission for extracted directories
	defaultDirPermission fs.FileMode

	// defaultFilePermission is the default file permission for extracted files
	defaultFilePermission fs.FileMode

	// denySymlinkExtraction offers the option to enable/disable the extraction of symlinks
	denySymlinkExtraction bool

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

	// telemetryHook is a function pointer to consume telemetry data after finished extraction
	// Important: do not adjust this value after extraction started
	telemetryHook telemetry.TelemetryHook

	// noUntarAfterDecompression offers the option to enable/disable combined tar.gz extraction
	noUntarAfterDecompression bool

	// Define if files should be overwritten in the destination
	overwrite bool

	// patterns is a list of file patterns to match files to extract
	patterns []string

	// verbose log extraction to stderr
	verbose bool
}

// NewConfig is a generator option that takes opts as adjustments of the
// default configuration in an option pattern style
func NewConfig(opts ...ConfigOption) *Config {
	const (
		cacheInMemory              = false
		continueOnError            = false
		continueOnUnsupportedFiles = false
		createDestination          = false
		defaultFilePermission      = 0640
		defaultDirPermission       = 0750
		denySymlinkExtraction      = false
		followSymlinks             = false
		maxFiles                   = 1000          // 1k files
		maxExtractionSize          = 1 << (10 * 3) // 1 Gb
		maxExtractionTime          = 60            // 1 minute
		maxInputSize               = 1 << (10 * 3) // 1 Gb
		noUntarAfterDecompression  = false
		overwrite                  = false
		verbose                    = false
	)

	// disable logging by default
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	// setup default values
	config := &Config{
		cacheInMemory:              cacheInMemory,
		continueOnError:            continueOnError,
		createDestination:          createDestination,
		defaultDirPermission:       defaultDirPermission,
		defaultFilePermission:      defaultFilePermission,
		denySymlinkExtraction:      denySymlinkExtraction,
		followSymlinks:             followSymlinks,
		logger:                     logger,
		maxFiles:                   maxFiles,
		maxExtractionSize:          maxExtractionSize,
		maxInputSize:               maxInputSize,
		overwrite:                  overwrite,
		noUntarAfterDecompression:  noUntarAfterDecompression,
		continueOnUnsupportedFiles: continueOnUnsupportedFiles,
		verbose:                    verbose,
	}

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}

	// return the modified house instance
	return config
}

// CacheInMemory returns true if caching in memory is enabled
func (c *Config) CacheInMemory() bool {
	return c.cacheInMemory
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

// ContinueOnError returns true if the extraction should continue on error
func (c *Config) ContinueOnError() bool {
	return c.continueOnError
}

// ContinueOnUnsupportedFiles returns true if unsupported files should be skipped
func (c *Config) ContinueOnUnsupportedFiles() bool {
	return c.continueOnUnsupportedFiles
}

// CreateDestination returns true if the destination directory should be created if it does not exist
func (c *Config) CreateDestination() bool {
	return c.createDestination
}

// DenySymlinkExtraction returns true if symlinks are NOT allowed
func (c *Config) DenySymlinkExtraction() bool {
	return c.denySymlinkExtraction
}

// DefaultDirPermission returns the default directory permission
func (c *Config) DefaultDirPermission() fs.FileMode {
	return c.defaultDirPermission
}

// DefaultFilePermission returns the default file permission
func (c *Config) DefaultFilePermission() fs.FileMode {
	return c.defaultFilePermission
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

// NoUntarAfterDecompression returns true if tar.gz should NOT be untarred after decompression
func (c *Config) NoUntarAfterDecompression() bool {
	return c.noUntarAfterDecompression
}

// Overwrite returns true if files should be overwritten in the destination
func (c *Config) Overwrite() bool {
	return c.overwrite
}

// Patterns returns a list of unix-filepath patterns to match files to extract
// Patterns are matched using [filepath.Match](https://golang.org/pkg/path/filepath/#Match).
func (c *Config) Patterns() []string {
	return c.patterns
}

// TelemetryHook returns the  telemetry hook
func (c *Config) TelemetryHook() telemetry.TelemetryHook {
	if c.telemetryHook == nil {
		return NoopTelemetryHook
	}
	return c.telemetryHook
}

// NoopTelemetryHook is a no operation telemetry hook
func NoopTelemetryHook(ctx context.Context, d *telemetry.Data) {
	// noop
}

// WithCacheInMemory options pattern function to enable/disable caching in memory.
// This applies only to the extraction of zip archives, which are provided as a stream.
func WithCacheInMemory(cache bool) ConfigOption {
	return func(c *Config) {
		c.cacheInMemory = cache
	}
}

// WithContinueOnError options pattern function to continue on error during extraction
func WithContinueOnError(yes bool) ConfigOption {
	return func(c *Config) {
		c.continueOnError = yes
	}
}

// WithContinueOnUnsupportedFiles options pattern function to enable/disable skipping unsupported files
func WithContinueOnUnsupportedFiles(ctd bool) ConfigOption {
	return func(c *Config) {
		c.continueOnUnsupportedFiles = ctd
	}
}

// WithCreateDestination options pattern function to create destination directory if it does not exist
func WithCreateDestination(create bool) ConfigOption {
	return func(c *Config) {
		c.createDestination = create
	}
}

// WithDenySymlinkExtraction options pattern function to deny symlink extraction
func WithDenySymlinkExtraction(deny bool) ConfigOption {
	return func(c *Config) {
		c.denySymlinkExtraction = deny
	}
}

// WithDefaultFilePermission options pattern function to set default file permission
func WithDefaultFilePermission(perm fs.FileMode) ConfigOption {
	return func(c *Config) {
		c.defaultFilePermission = perm
	}
}

// WithDefaultDirPermission options pattern function to set default directory permission
func WithDefaultDirPermission(perm fs.FileMode) ConfigOption {
	return func(c *Config) {
		c.defaultDirPermission = perm
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

// WithMaxExtractionSize options pattern function to set WithMaxExtractionSize in the
// config (-1 to disable check)
func WithMaxExtractionSize(maxExtractionSize int64) ConfigOption {
	return func(c *Config) {
		c.maxExtractionSize = maxExtractionSize
	}
}

// WithMaxFiles options pattern function to set maxFiles in the config (-1 to disable check)
func WithMaxFiles(maxFiles int64) ConfigOption {
	return func(c *Config) {
		c.maxFiles = maxFiles
	}
}

// WithMaxInputSize options pattern function to set MaxInputSize in the
// config (-1 to disable check)
func WithMaxInputSize(maxInputSize int64) ConfigOption {
	return func(c *Config) {
		c.maxInputSize = maxInputSize
	}
}

// WithNoUntarAfterDecompression options pattern function to enable/disable combined tar.gz extraction
func WithNoUntarAfterDecompression(disable bool) ConfigOption {
	return func(c *Config) {
		c.noUntarAfterDecompression = disable
	}
}

// WithOverwrite options pattern function to set overwrite in the config
func WithOverwrite(enable bool) ConfigOption {
	return func(c *Config) {
		c.overwrite = enable
	}
}

// WithPatterns options pattern function to set filepath pattern
func WithPatterns(pattern ...string) ConfigOption {
	return func(c *Config) {
		c.patterns = append(c.patterns, pattern...)
	}
}

// WithTelemetryHook options pattern function to set a telemetry hook
func WithTelemetryHook(hook telemetry.TelemetryHook) ConfigOption {
	return func(c *Config) {
		c.telemetryHook = hook
	}
}
