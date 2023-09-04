package config

import (
	"fmt"
	"testing"
)

// TestCheckMaxFiles implements test cases
func TestCheckMaxFiles(t *testing.T) {
	// prepare testcases
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

	// prepare testcases
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
			name:        "disable filzes check",
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

	// prepare testcases
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

// TestCheckWithMaxExtractionTime implements test cases
func TestCheckWithMaxExtractionTime(t *testing.T) {

	// prepare testcases
	cases := []struct {
		name   string
		config *Config
		expect int64
	}{
		{
			name:   "Check for 5 second timeout",
			config: NewConfig(WithMaxExtractionTime(5)), // enable overwrite
			expect: 5,
		},
		{
			name:   "Check disabled timeout",
			config: NewConfig(WithMaxExtractionTime(-1)), // disable overwrite
			expect: -1,
		},
		{
			name:   "Default is disabled",
			config: NewConfig(), // check default value
			expect: 60,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expect
			got := tc.config.MaxExtractionTime
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

// TestCheckWithOverwrite implements test cases
func TestCheckWithDenySymlinks(t *testing.T) {

	// prepare testcases
	cases := []struct {
		name   string
		config *Config
		expect bool
	}{
		{
			name:   "Allow symlinks",
			config: NewConfig(WithDenySymlinks(false)), // enable overwrite
			expect: false,
		},
		{
			name:   "Deny symlinks",
			config: NewConfig(WithDenySymlinks(true)), // disable overwrite
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
			got := tc.config.DenySymlinks
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

// TestCheckWithOverwrite implements test cases
func TestCheckWithContinueOnError(t *testing.T) {

	// prepare testcases
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
			name:   "Dont continue on error",
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

	// prepare testcases
	cases := []struct {
		name   string
		config *Config
		expect bool
	}{
		{
			name:   "Dont follow symlinks",
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

// TestCheckWithFollowSymlinks implements test cases
func TestCheckWithVerbose(t *testing.T) {

	// prepare testcases
	cases := []struct {
		name   string
		config *Config
		expect bool
	}{
		{
			name:   "Not verbose",
			config: NewConfig(WithVerbose(false)),
			expect: false,
		},
		{
			name:   "Be verbose",
			config: NewConfig(WithVerbose(true)),
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
			got := tc.config.Verbose
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}
