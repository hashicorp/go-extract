package extractor

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// TestCreateFile is a wrapper around the createFile function
func TestCreateFile(t *testing.T) {

	tc := []struct {
		dst         string
		name        string
		src         string
		mode        fs.FileMode
		maxSize     int64
		cfg         *config.Config
		expectError bool
		prep        func(target.Target, string)
	}{
		{
			name:    "test.txt",
			src:     "Hello, World!",
			mode:    0640,
			maxSize: -1,
		},
		{
			name:        "",
			src:         "Hello, World!",
			mode:        0640,
			maxSize:     -1,
			expectError: true,
		},
		{
			dst:     "test",
			name:    "test.txt",
			src:     "Hello, World!",
			mode:    0640,
			maxSize: -1,
			cfg:     config.NewConfig(config.WithCreateDestination(true)),
		},
		{
			dst:     "foo/bar",
			name:    "test.txt",
			src:     "Hello, World!",
			mode:    0640,
			maxSize: -1,
			cfg:     config.NewConfig(config.WithCreateDestination(true)),
			prep: func(t target.Target, dst string) {
				t.CreateDir(filepath.Join(dst, "foo"), 0000)
			},
			expectError: true,
		},
		{
			dst:         "foo",
			name:        "test.txt",
			src:         "Hello, World!",
			mode:        0640,
			maxSize:     -1,
			cfg:         config.NewConfig(config.WithCreateDestination(false)),
			expectError: true,
		},
	}

	for i, tt := range tc {

		var testTarget = target.NewNoopTarget()

		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(testTarget, tt.dst)
		}
		_, err := createFile(testTarget, tt.dst, tt.name, strings.NewReader(tt.src), tt.mode, tt.maxSize, tt.cfg)
		if tt.expectError != (err != nil) {
			t.Errorf("[%v] createFile(%s, %s, %s, %d, %d) = %v; want nil", i, tt.dst, tt.name, tt.src, tt.mode, tt.maxSize, err)
		}
	}
}

// TestCreateDir implements tests for the createDir function
func TestCreateDir(t *testing.T) {

	tc := []struct {
		dst         string
		name        string
		mode        fs.FileMode
		cfg         *config.Config
		expectError bool
		prep        func(target.Target, string)
	}{
		{
			name: "test",
			mode: 0750,
		},
		{
			name:        "",
			mode:        0750,
			expectError: true,
		},
		{
			dst:  "foo",
			name: "bar",
			mode: 0750,
			cfg:  config.NewConfig(config.WithCreateDestination(true)),
		},
		{
			dst:         "foo",
			name:        "bar",
			mode:        0750,
			cfg:         config.NewConfig(config.WithCreateDestination(false)),
			expectError: true,
		},
		{
			dst:  "foo",
			name: "bar",
			mode: 0750,
			cfg:  config.NewConfig(config.WithCreateDestination(true)),
			prep: func(t target.Target, dst string) {
				t.CreateDir(filepath.Join(dst, "foo"), 0000)
			},
			expectError: true,
		},
		{
			dst:         "",
			name:        "/failingt-extract",
			mode:        0750,
			expectError: true,
		},
	}

	for i, tt := range tc {

		testTarget := target.NewNoopTarget()
		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(testTarget, tt.dst)
		}
		err := createDir(testTarget, tt.dst, tt.name, tt.mode, tt.cfg)
		gotError := (err != nil)
		if tt.expectError != gotError {
			t.Errorf("[%v] createDir(dst=%s, name=%s, mode=%o, createDest=%v, defaultDirPerm=%o) = ERROR(%v); want %v", i, tt.dst, tt.name, tt.mode.Perm(), tt.cfg.CreateDestination(), tt.cfg.CustomCreateDirMode(), err, tt.expectError)
		}
	}
}

// TestCreateSymlink implements tests for the createSymlink function
func TestCreateSymlink(t *testing.T) {

	tc := []struct {
		dst         string
		name        string
		linkTarget  string
		cfg         *config.Config
		expectError bool
		prep        func(target.Target, string)
	}{
		{
			name:       "test", // 0
			linkTarget: "test.txt",
		},
		{
			name:        "", // 1
			linkTarget:  "test.txt",
			expectError: true,
		},
		{
			dst:         "foo", // 2
			name:        "bar",
			linkTarget:  "test.txt",
			cfg:         config.NewConfig(config.WithDenySymlinkExtraction(false)),
			expectError: true,
		},
		{
			dst:         "foo", // 3
			name:        "bar",
			linkTarget:  "test.txt",
			cfg:         config.NewConfig(config.WithCreateDestination(true), config.WithDenySymlinkExtraction(false)),
			expectError: false,
		},
		{
			dst:         "do-not-allow", // 4
			name:        "bar",
			linkTarget:  "test.txt",
			cfg:         config.NewConfig(config.WithDenySymlinkExtraction(true), config.WithContinueOnError(false)),
			expectError: true,
		},
		{
			dst:         "do-not-allow", // 5
			name:        "bar",
			linkTarget:  "test.txt",
			cfg:         config.NewConfig(config.WithDenySymlinkExtraction(true), config.WithContinueOnError(true)),
			expectError: false,
		},
		{
			dst:         "foo", // 6
			name:        "bar",
			linkTarget:  "/test.txt",
			cfg:         config.NewConfig(config.WithDenySymlinkExtraction(false)),
			expectError: true,
		},
		{
			dst:         "foo", // 7
			name:        "bar",
			linkTarget:  "/test.txt",
			cfg:         config.NewConfig(config.WithDenySymlinkExtraction(false), config.WithContinueOnError(true)),
			expectError: false,
		},
		{
			dst:         "foo", // 8
			name:        "bar",
			linkTarget:  "/test.txt",
			cfg:         config.NewConfig(config.WithDenySymlinkExtraction(true)),
			expectError: true,
		},
	}

	for i, tt := range tc {

		testTarget := target.NewNoopTarget()
		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(testTarget, tt.dst)
		}
		err := createSymlink(testTarget, tt.dst, tt.name, tt.linkTarget, tt.cfg)
		gotError := (err != nil)
		if tt.expectError != gotError {
			t.Errorf("[%v] createSymlink(dst=%s, name=%s, linkTarget=%s, denySymlinkExtraction=%v) = ERROR(%v); want %v", i, tt.dst, tt.name, tt.linkTarget, tt.cfg.DenySymlinkExtraction(), err, tt.expectError)
		}
	}
}
