package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/hashicorp/go-extract/metrics"
)

// TestCheckMaxFiles implements test cases
func TestCheckMaxFiles(t *testing.T) {
	// prepare test cases
	cases := []struct {
		name        string
		input       int64
		config      *Config
		expectError bool
	}{
		{
			name:        "less files then maximum",
			input:       5,                           // within limit
			config:      NewConfig(WithMaxFiles(10)), // 10
			expectError: false,
		},
		{
			name:        "more files then maximum",
			input:       15,                          // over limit
			config:      NewConfig(WithMaxFiles(10)), // 10
			expectError: true,
		},
		{
			name:        "disable file counter check",
			input:       5000,                        // ignored
			config:      NewConfig(WithMaxFiles(-1)), // disable
			expectError: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expectError
			got := tc.config.CheckMaxObjects(tc.input) != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

// TestWithMaxFiles implements test cases
func TestWithMaxInputSize(t *testing.T) {
	maxInputSize := int64(1024)
	config := &Config{}
	option := WithMaxInputSize(maxInputSize)
	option(config)

	if config.MaxInputSize() != maxInputSize {
		t.Errorf("Expected MaxInputSize to be %d, but got %d", maxInputSize, config.MaxInputSize())
	}
}

func TestContinueOnUnsupportedFiles(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "continueOnUnsupportedFiles is true",
			cfg:  &Config{continueOnUnsupportedFiles: true},
			want: true,
		},
		{
			name: "continueOnUnsupportedFiles is false",
			cfg:  &Config{continueOnUnsupportedFiles: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ContinueOnUnsupportedFiles(); got != tt.want {
				t.Errorf("ContinueOnUnsupportedFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithPattern(t *testing.T) {
	patterns := []string{"test1", "test2"}
	cfg := NewConfig(WithPatterns(patterns...))

	if len(cfg.patterns) != len(patterns) {
		t.Errorf("WithPattern() = %v, want %v", len(cfg.patterns), len(patterns))
	}

	for i, p := range patterns {
		if cfg.Patterns()[i] != p {
			t.Errorf("WithPattern() pattern = %v, want %v", cfg.patterns[i], p)
		}
	}
}

func TestWithMaxExtractionSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want int64
	}{
		{
			name: "Set max extraction size to 100",
			size: 100,
			want: 100,
		},
		{
			name: "Set max extraction size to -1 (disable check)",
			size: -1,
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithMaxExtractionSize(tt.size)
			option(config)

			if config.maxExtractionSize != tt.want {
				t.Errorf("WithMaxExtractionSize() set maxExtractionSize to %v, want %v", config.maxExtractionSize, tt.want)
			}
		})
	}
}

func TestCacheInMemory(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "cacheInMemory is true",
			cfg:  &Config{cacheInMemory: true},
			want: true,
		},
		{
			name: "cacheInMemory is false",
			cfg:  &Config{cacheInMemory: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.CacheInMemory(); got != tt.want {
				t.Errorf("CacheInMemory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoTarGzExtract(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "noUntarAfterDecompression is true",
			cfg:  &Config{noUntarAfterDecompression: true},
			want: true,
		},
		{
			name: "noUntarAfterDecompression is false",
			cfg:  &Config{noUntarAfterDecompression: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.NoUntarAfterDecompression(); got != tt.want {
				t.Errorf("noUntarAfterDecompression() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithContinueOnUnsupportedFiles(t *testing.T) {
	tests := []struct {
		name string
		ctd  bool
		want bool
	}{
		{
			name: "Enable continue on unsupported files",
			ctd:  true,
			want: true,
		},
		{
			name: "Disable continue on unsupported files",
			ctd:  false,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithContinueOnUnsupportedFiles(tt.ctd)
			option(config)

			if config.continueOnUnsupportedFiles != tt.want {
				t.Errorf("WithContinueOnUnsupportedFiles() set continueOnUnsupportedFiles to %v, want %v", config.continueOnUnsupportedFiles, tt.want)
			}
		})
	}
}

func TestWithCacheInMemory(t *testing.T) {
	tests := []struct {
		name  string
		cache bool
		want  bool
	}{
		{
			name:  "Enable cache in memory",
			cache: true,
			want:  true,
		},
		{
			name:  "Disable cache in memory",
			cache: false,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithCacheInMemory(tt.cache)
			option(config)

			if config.cacheInMemory != tt.want {
				t.Errorf("WithCacheInMemory() set cacheInMemory to %v, want %v", config.cacheInMemory, tt.want)
			}
		})
	}
}

func TestWithNoTarGzExtract(t *testing.T) {
	tests := []struct {
		name     string
		disabled bool
		want     bool
	}{
		{
			name:     "Disable tar.gz extraction",
			disabled: true,
			want:     true,
		},
		{
			name:     "Enable tar.gz extraction",
			disabled: false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithNoUntarAfterDecompression(tt.disabled)
			option(config)

			if config.noUntarAfterDecompression != tt.want {
				t.Errorf("WithUntarAfterDecompression() set noUntarAfterDecompression to %v, want %v", config.noUntarAfterDecompression, tt.want)
			}
		})
	}
}

// TestWithLogger implements test cases
func TestWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	config := &Config{}
	option := WithLogger(logger)
	option(config)

	if config.Logger() == nil {
		t.Errorf("Expected Logger to be set, but it was nil")
	}
}

// TestCheckMaxObjects implements test cases
func TestCheckMaxObjects(t *testing.T) {
	config := &Config{maxFiles: 5}

	err := config.CheckMaxObjects(6)
	if err == nil {
		t.Errorf("Expected error when counter exceeds MaxFiles, but got nil")
	}

	err = config.CheckMaxObjects(5)
	if err != nil {
		t.Errorf("Expected no error when counter equals MaxFiles, but got: %s", err)
	}

	err = config.CheckMaxObjects(4)
	if err != nil {
		t.Errorf("Expected no error when counter is less than MaxFiles, but got: %s", err)
	}

	config.maxFiles = -1
	err = config.CheckMaxObjects(6)
	if err != nil {
		t.Errorf("Expected no error when MaxFiles is -1, but got: %s", err)
	}
}

// TestCheckExtractionSize implements test cases
func TestCheckExtractionSize(t *testing.T) {
	config := &Config{maxExtractionSize: 1024}

	err := config.CheckExtractionSize(2048)
	if err == nil {
		t.Errorf("Expected error when fileSize exceeds MaxExtractionSize, but got nil")
	}

	err = config.CheckExtractionSize(1024)
	if err != nil {
		t.Errorf("Expected no error when fileSize equals MaxExtractionSize, but got: %s", err)
	}

	err = config.CheckExtractionSize(512)
	if err != nil {
		t.Errorf("Expected no error when fileSize is less than MaxExtractionSize, but got: %s", err)
	}

	config.maxExtractionSize = -1
	err = config.CheckExtractionSize(2048)
	if err != nil {
		t.Errorf("Expected no error when MaxExtractionSize is -1, but got: %s", err)
	}
}

// TestWithCreateDestination implements test cases
func TestWithCreateDestination(t *testing.T) {
	config := &Config{}
	option := WithCreateDestination(true)
	option(config)

	if config.CreateDestination() != true {
		t.Errorf("Expected CreateDestination to be true, but got false")
	}

	option = WithCreateDestination(false)
	option(config)

	if config.CreateDestination() != false {
		t.Errorf("Expected CreateDestination to be false, but got true")
	}
}

// TestCheckWithOverwrite implements test cases
func TestCheckWithOverwrite(t *testing.T) {

	// prepare test cases
	cases := []struct {
		name   string
		config *Config
		expect bool
	}{
		{
			name:   "Overwrite enabled",
			config: NewConfig(WithOverwrite(true)), // enable overwrite
			expect: true,
		},
		{
			name:   "Overwrite disabled",
			config: NewConfig(WithOverwrite(false)), // disable overwrite
			expect: false,
		},
		{
			name:   "Default is disabled",
			config: NewConfig(), // check default value
			expect: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expect
			got := tc.config.Overwrite()
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

// TestCheckWithOverwrite implements test cases
func TestCheckWithDenySymlinks(t *testing.T) {

	// prepare test cases
	cases := []struct {
		name   string
		config *Config
		expect bool
	}{
		{
			name:   "Allow symlinks",
			config: NewConfig(WithDenySymlinkExtraction(false)), // disable symlinks
			expect: false,
		},
		{
			name:   "Deny symlinks",
			config: NewConfig(WithDenySymlinkExtraction(true)), // allow symlinks
			expect: true,
		},
		{
			name:   "Default is disabled",
			config: NewConfig(), // check default value
			expect: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expect
			got := tc.config.DenySymlinkExtraction()
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

// TestCheckWithOverwrite implements test cases
func TestCheckWithContinueOnError(t *testing.T) {

	// prepare test cases
	cases := []struct {
		name   string
		config *Config
		expect bool
	}{
		{
			name:   "Do continue on error",
			config: NewConfig(WithContinueOnError(true)), // enable overwrite
			expect: true,
		},
		{
			name:   "Don't continue on error",
			config: NewConfig(WithContinueOnError(false)), // disable overwrite
			expect: false,
		},
		{
			name:   "Default is disabled",
			config: NewConfig(), // check default value
			expect: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expect
			got := tc.config.ContinueOnError()
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

// TestCheckWithFollowSymlinks implements test cases
func TestCheckWithFollowSymlinks(t *testing.T) {

	// prepare test cases
	cases := []struct {
		name   string
		config *Config
		expect bool
	}{
		{
			name:   "Don't follow symlinks",
			config: NewConfig(WithFollowSymlinks(false)),
			expect: false,
		},
		{
			name:   "Follow symlinks",
			config: NewConfig(WithFollowSymlinks(true)),
			expect: true,
		},
		{
			name:   "Default is disabled",
			config: NewConfig(), // check default value
			expect: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expect
			got := tc.config.FollowSymlinks()
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

func TestWithMetricsHook(t *testing.T) {

	// Create a new Config without specified hook
	metricsDelivered := false
	c := NewConfig()

	// submit hook
	c.MetricsHook()(context.Background(), &metrics.Metrics{})

	// check if hook was delivered
	if metricsDelivered {
		t.Errorf("Expected no operation, but metricsDelivered has changed")
	}

	// Create a new Config with specified hook
	c = NewConfig(WithMetricsHook(func(ctx context.Context, m *metrics.Metrics) {
		metricsDelivered = true
	}))

	// submit hook
	c.MetricsHook()(context.Background(), &metrics.Metrics{})

	// check if hook was delivered
	if !metricsDelivered {
		t.Errorf("Expected metrics value to be delivered, but it was not")
	}

}
