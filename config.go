// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
)

// ConfigOption is a function pointer to implement the option pattern
type ConfigOption func(*Config)

// Config provides a configuration struct and options to adjust the configuration.
//
// The configuration struct holds all configuration options for the extraction process.
// The configuration options can be adjusted using the option pattern style.
//
// The default configuration is designed to be secure by default and prevent exhaustion,
// path traversal and symlink attacks.
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

	// customCreateDirMode is the file mode for created directories, that are not defined in the archive (respecting umask)
	customCreateDirMode fs.FileMode

	// customDecompressFileMode is the file mode for a decompressed file (respecting umask)
	customDecompressFileMode fs.FileMode

	// denySymlinkExtraction offers the option to enable/disable the extraction of symlinks
	denySymlinkExtraction bool

	// dropFileAttributes is a flag drop the file attributes of the extracted files
	dropFileAttributes bool

	// extractionType is the type of extraction algorithm
	extractionType string

	// traverseSymlinks traverses symlinks to directories during extraction
	traverseSymlinks bool

	// logger stream for extraction
	logger logger

	// maxExtractionSize is the maximum size of a file after decompression.
	// Set value to -1 to disable the check.
	maxExtractionSize int64

	// maxFiles is the maximum of files (including folder and symlinks) in an archive.
	// Set value to -1 to disable the check.
	maxFiles int64

	// maxInputSize is the maximum size of the input
	// Set value to -1 to disable the check.
	maxInputSize int64

	// telemetryHook is a function to consume telemetry data after finished extraction
	// Important: do not adjust this value after extraction started
	telemetryHook TelemetryHook

	// noUntarAfterDecompression offers the option to enable/disable combined tar.gz extraction
	noUntarAfterDecompression bool

	// Define if files should be overwritten in the destination
	overwrite bool

	// patterns is a list of file patterns to match files to extract
	patterns []string

	// preserveOwner is a flag to preserve the owner of the extracted files
	preserveOwner bool
}

// ContinueOnError returns true if the extraction should continue on error.
func (c *Config) ContinueOnError() bool {
	return c.continueOnError
}

// CacheInMemory returns true if caching in memory is enabled. This applies only to
// the extraction of zip archives, which are provided as a stream.
//
// If set to false, the cache is stored on disk to avoid memory exhaustion.
func (c *Config) CacheInMemory() bool {
	return c.cacheInMemory
}

// CheckMaxFiles checks if counter exceeds the configured maximum. If the maximum is exceeded,
// a [ErrMaxFilesExceeded] error is returned.
func (c *Config) CheckMaxFiles(counter int64) error {

	// check if disabled
	if c.MaxFiles() == -1 {
		return nil
	}

	// check value
	if counter > c.MaxFiles() {
		return ErrMaxFilesExceeded
	}
	return nil
}

// CheckExtractionSize checks if fileSize exceeds configured maximum. If the maximum is exceeded,
// a [ErrMaxExtractionSizeExceeded] error is returned.
func (c *Config) CheckExtractionSize(fileSize int64) error {

	// check if disabled
	if c.MaxExtractionSize() == -1 {
		return nil
	}

	// check value
	if fileSize > c.MaxExtractionSize() {
		return ErrMaxExtractionSizeExceeded
	}
	return nil
}

// ContinueOnUnsupportedFiles returns true if unsupported files, e.g., FIFO, block or
// character devices, should be skipped.
//
// If symlinks are not allowed and a symlink is found, it is considered an unsupported
// file.
func (c *Config) ContinueOnUnsupportedFiles() bool {
	return c.continueOnUnsupportedFiles
}

// CreateDestination returns true if the destination directory should be
// created if it does not exist.
func (c *Config) CreateDestination() bool {
	return c.createDestination
}

// CustomCreateDirMode returns the file mode for created directories,
// that are not defined in the archive. (respecting umask)
func (c *Config) CustomCreateDirMode() fs.FileMode {
	return c.customCreateDirMode
}

// CustomDecompressFileMode returns the file mode for a decompressed file.
// (respecting umask)
func (c *Config) CustomDecompressFileMode() fs.FileMode {
	return c.customDecompressFileMode
}

// DenySymlinkExtraction returns true if symlinks are NOT allowed.
func (c *Config) DenySymlinkExtraction() bool {
	return c.denySymlinkExtraction
}

// DropFileAttributes returns true if the file attributes should be dropped.
func (c *Config) DropFileAttributes() bool {
	return c.dropFileAttributes
}

// ExtractType returns the specified extraction type.
func (c *Config) ExtractType() string {
	return c.extractionType
}

// TraverseSymlinks returns true if symlinks should be traversed during extraction.
func (c *Config) TraverseSymlinks() bool {
	return c.traverseSymlinks
}

// Logger returns the logger.
func (c *Config) Logger() logger {
	return c.logger
}

// MaxExtractionSize returns the maximum size over all decompressed and extracted files.
func (c *Config) MaxExtractionSize() int64 {
	return c.maxExtractionSize
}

// MaxFiles returns the maximum of files (including folder and symlinks) in an archive.
func (c *Config) MaxFiles() int64 {
	return c.maxFiles
}

// MaxInputSize returns the maximum size of the input.
func (c *Config) MaxInputSize() int64 {
	return c.maxInputSize
}

// NoUntarAfterDecompression returns true if tar.gz should NOT be untared after decompression.
func (c *Config) NoUntarAfterDecompression() bool {
	return c.noUntarAfterDecompression
}

// Overwrite returns true if files should be overwritten in the destination.
func (c *Config) Overwrite() bool {
	return c.overwrite
}

// Patterns returns a list of unix-filepath patterns to match files to extract
// Patterns are matched using [filepath.Match](https://golang.org/pkg/path/filepath/#Match).
func (c *Config) Patterns() []string {
	return c.patterns
}

// PreserveOwner returns true if the owner of the extracted files should
// be preserved. This option is only available on Unix systems requiring
// root privileges and tar archives as input.
func (c *Config) PreserveOwner() bool {
	return c.preserveOwner
}

// SetNoUntarAfterDecompression sets the noUntarAfterDecompression flag. If true, tar.gz files
// are not untared after decompression.
func (c *Config) SetNoUntarAfterDecompression(b bool) {
	c.noUntarAfterDecompression = b
}

// TelemetryHook returns the  telemetry hook.
func (c *Config) TelemetryHook() TelemetryHook {
	if c.telemetryHook == nil {
		return func(ctx context.Context, d *TelemetryData) {
			// noop
		}
	}
	return c.telemetryHook
}

const (
	defaultCacheInMemory              = false         // cache on disk
	defaultContinueOnError            = false         // stop on error and return error
	defaultContinueOnUnsupportedFiles = false         // stop on unsupported files and return error
	defaultCreateDestination          = false         // don't create destination directory
	defaultCustomCreateDirMode        = 0750          // default directory permissions rwxr-x---
	defaultCustomDecompressFileMode   = 0640          // default decompression permissions rw-r-----
	defaultDenySymlinkExtraction      = false         // allow symlink extraction
	defaultDropFileAttributes         = false         // drop file attributes from archive
	defaultExtractionType             = ""            // don't limit extraction type
	defaultMaxFiles                   = 100000        // 100k files
	defaultMaxExtractionSize          = 1 << (10 * 3) // 1 Gb
	defaultMaxInputSize               = 1 << (10 * 3) // 1 Gb
	defaultNoUntarAfterDecompression  = false         // untar after decompression
	defaultOverwrite                  = false         // don't overwrite existing files
	defaultPreserveOwner              = false         // don't preserve owner
	defaultTraverseSymlinks           = false         // don't traverse symlinks

)

var (
	// slog to discard
	defaultLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	// no operation telemetry hook
	defaultTelemetryHook = func(ctx context.Context, d *TelemetryData) {
		// noop
	}
)

// NewConfig is a generator option that takes opts as adjustments of the
// default configuration in an option pattern style.
func NewConfig(opts ...ConfigOption) *Config {

	// setup default values
	config := &Config{
		cacheInMemory:              defaultCacheInMemory,
		continueOnError:            defaultContinueOnError,
		continueOnUnsupportedFiles: defaultContinueOnUnsupportedFiles,
		createDestination:          defaultCreateDestination,
		customCreateDirMode:        defaultCustomCreateDirMode,
		customDecompressFileMode:   defaultCustomDecompressFileMode,
		denySymlinkExtraction:      defaultDenySymlinkExtraction,
		dropFileAttributes:         defaultDropFileAttributes,
		extractionType:             defaultExtractionType,
		logger:                     defaultLogger,
		maxFiles:                   defaultMaxFiles,
		maxExtractionSize:          defaultMaxExtractionSize,
		maxInputSize:               defaultMaxInputSize,
		overwrite:                  defaultOverwrite,
		telemetryHook:              defaultTelemetryHook,
		traverseSymlinks:           defaultTraverseSymlinks,
		noUntarAfterDecompression:  defaultNoUntarAfterDecompression,
		preserveOwner:              defaultPreserveOwner,
	}

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}

	// return the modified house instance
	return config
}

// WithCacheInMemory options pattern function to enable/disable caching in memory.
// This applies only to the extraction of zip archives, which are provided as a stream.
//
// If set to false, the cache is stored on disk to avoid memory exhaustion.
func WithCacheInMemory(cache bool) ConfigOption {
	return func(c *Config) {
		c.cacheInMemory = cache
	}
}

// WithContinueOnError options pattern function to continue on error during extraction. If set to true,
// the error is logged and the extraction continues. If set to false, the extraction stops and returns the error.
func WithContinueOnError(yes bool) ConfigOption {
	return func(c *Config) {
		c.continueOnError = yes
	}
}

// WithContinueOnUnsupportedFiles options pattern function to
// enable/disable skipping unsupported files. An unsupported file is a file
// that is not supported by the extraction algorithm. If symlinks are not allowed
// and a symlink is found, it is considered an unsupported file.
func WithContinueOnUnsupportedFiles(ctd bool) ConfigOption {
	return func(c *Config) {
		c.continueOnUnsupportedFiles = ctd
	}
}

// WithCreateDestination options pattern function to create
// destination directory if it does not exist.
func WithCreateDestination(create bool) ConfigOption {
	return func(c *Config) {
		c.createDestination = create
	}
}

// WithCustomCreateDirMode options pattern function to set the file mode
// for created directories, that are not defined in the archive. (respecting umask)
func WithCustomCreateDirMode(mode fs.FileMode) ConfigOption {
	return func(c *Config) {
		c.customCreateDirMode = mode
	}
}

// WithCustomDecompressFileMode options pattern function to set the file mode for a
// decompressed file. (respecting umask)
func WithCustomDecompressFileMode(mode fs.FileMode) ConfigOption {
	return func(c *Config) {
		c.customDecompressFileMode = mode
	}
}

// WithDenySymlinkExtraction options pattern function to deny symlink extraction.
func WithDenySymlinkExtraction(deny bool) ConfigOption {
	return func(c *Config) {
		c.denySymlinkExtraction = deny
	}
}

// WithDropFileAttributes options pattern function to drop the
// file attributes of the extracted files.
func WithDropFileAttributes(drop bool) ConfigOption {
	return func(c *Config) {
		c.dropFileAttributes = drop
	}
}

// WithExtractType options pattern function to set the extraction type in the [Config].
func WithExtractType(extractionType string) ConfigOption {
	return func(c *Config) {
		if len(extractionType) > 0 {
			c.extractionType = extractionType
		}
	}
}

// WithInsecureTraverseSymlinks options pattern function to traverse symlinks during extraction.
func WithInsecureTraverseSymlinks(traverse bool) ConfigOption {
	return func(c *Config) {
		c.traverseSymlinks = traverse
	}
}

// WithLogger options pattern function to set a custom logger.
func WithLogger(logger logger) ConfigOption {
	return func(c *Config) {
		c.logger = logger
	}
}

// WithMaxExtractionSize options pattern function to set maximum size over all decompressed
//
//	and extracted files. (-1 to disable check)
func WithMaxExtractionSize(maxExtractionSize int64) ConfigOption {
	return func(c *Config) {
		c.maxExtractionSize = maxExtractionSize
	}
}

// WithMaxFiles options pattern function to set maximum number of extracted, files, directories
// and symlinks during the extraction. (-1 to disable check)
func WithMaxFiles(maxFiles int64) ConfigOption {
	return func(c *Config) {
		c.maxFiles = maxFiles
	}
}

// WithMaxInputSize options pattern function to set MaxInputSize for extraction input file. (-1 to disable check)
func WithMaxInputSize(maxInputSize int64) ConfigOption {
	return func(c *Config) {
		c.maxInputSize = maxInputSize
	}
}

// WithNoUntarAfterDecompression options pattern function to enable/disable combined tar.gz extraction.
func WithNoUntarAfterDecompression(disable bool) ConfigOption {
	return func(c *Config) {
		c.noUntarAfterDecompression = disable
	}
}

// WithOverwrite options pattern function specify if files should be overwritten in the destination.
func WithOverwrite(enable bool) ConfigOption {
	return func(c *Config) {
		c.overwrite = enable
	}
}

// WithPatterns options pattern function to set filepath pattern, that files need to match to be extracted.
// Patterns are matched using [pkg/path/filepath.Match].
func WithPatterns(pattern ...string) ConfigOption {
	return func(c *Config) {
		c.patterns = append(c.patterns, pattern...)
	}
}

// WithPreserveOwner options pattern function to preserve the owner of
// the extracted files. This option is only available on Unix systems
// requiring root privileges and tar archives as input.
func WithPreserveOwner(preserve bool) ConfigOption {
	return func(c *Config) {
		c.preserveOwner = preserve
	}
}

// WithTelemetryHook options pattern function to set a [telemetry.TelemetryHook], which is called after extraction.
func WithTelemetryHook(hook TelemetryHook) ConfigOption {
	return func(c *Config) {
		c.telemetryHook = hook
	}
}
