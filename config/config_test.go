package config

import (
	"fmt"
	"testing"
)

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

func TestCheckFileSize(t *testing.T) {

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
