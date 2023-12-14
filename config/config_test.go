package config

import (
	"fmt"
	"testing"
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
			got := tc.config.CheckMaxFiles(tc.input) != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

// TestCheckExtractionSize implements test cases
func TestCheckExtractionSize(t *testing.T) {

	// prepare test cases
	cases := []struct {
		name        string
		input       int64
		config      *Config
		expectError bool
	}{
		{
			name:        "file size less then maximum",
			input:       1 << (9 * 1),                                    // 512b
			config:      NewConfig(WithMaxExtractionSize(1 << (10 * 1))), // 1kb
			expectError: false,
		},
		{
			name:        "file bigger then maximum",
			input:       5 << (10 * 1),                                   // 5kb
			config:      NewConfig(WithMaxExtractionSize(1 << (10 * 1))), // 1kb
			expectError: true,
		},
		{
			name:        "disable file size check",
			input:       5 << (10 * 1),                        // 5kb
			config:      NewConfig(WithMaxExtractionSize(-1)), // 1kb
			expectError: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expectError
			got := tc.config.CheckExtractionSize(tc.input) != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
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
			got := tc.config.Overwrite
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
			config: NewConfig(WithAllowSymlinks(false)), // disable symlinks
			expect: false,
		},
		{
			name:   "Deny symlinks",
			config: NewConfig(WithAllowSymlinks(true)), // allow symlinks
			expect: true,
		},
		{
			name:   "Default is enabled",
			config: NewConfig(), // check default value
			expect: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expect
			got := tc.config.AllowSymlinks
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
			got := tc.config.ContinueOnError
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
			got := tc.config.FollowSymlinks
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}
