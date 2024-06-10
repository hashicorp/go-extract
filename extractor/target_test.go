package extractor

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
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
				if err := t.CreateDir(filepath.Join(dst, "foo"), 0000); err != nil {
					panic(fmt.Errorf("failed to create dir: %s", err))
				}
			},
			expectError: runtime.GOOS != "windows", // only relevant test for unix based systems
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

		var testTarget = target.NewOS()
		tmpDir := t.TempDir()

		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(testTarget, tmpDir)
		}
		dst := filepath.Join(tmpDir, tt.dst)
		_, err := createFile(testTarget, dst, tt.name, strings.NewReader(tt.src), tt.mode, tt.maxSize, tt.cfg)
		if tt.expectError != (err != nil) {
			t.Errorf("[%v] createFile(%s, %s, %s, %d, %d) = %v; want %v", i, tt.dst, tt.name, tt.src, tt.mode, tt.maxSize, err, tt.expectError)
		}
	}
}

// TestCreateDir implements tests for the createDir function
func TestCreateDir(t *testing.T) {

	tc := []struct {
		dst           string
		name          string
		mode          fs.FileMode
		cfg           *config.Config
		expectError   bool
		prep          func(target.Target, string)
		dontConcatDst bool
	}{
		{
			name: "test",
			mode: 0750,
		},
		{
			name:        "",
			mode:        0750,
			expectError: false,
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
				if err := t.CreateDir(filepath.Join(dst, "foo"), 0000); err != nil {
					panic(fmt.Errorf("failed to create dir: %s", err))
				}
			},
			expectError: (runtime.GOOS != "windows"), // only relevant test for unix based systems
		},
		{
			dst:         "",
			name:        "/failingt-extract",
			mode:        0750,
			expectError: false, // bc, name is concatenated with tmpDir
		},
		{
			dst:           "",
			name:          "/failingt-extract",
			mode:          0750,
			expectError:   true, // bc, name is *not* concatenated with tmpDir
			dontConcatDst: true,
		},
		{
			dst:           "",
			name:          "./failingt-extract",
			mode:          0750,
			expectError:   false,
			dontConcatDst: true,
		},
	}

	for i, tt := range tc {

		testTarget := target.NewOS()
		tmpDir := t.TempDir()

		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(testTarget, tmpDir)
		}
		dst := tt.dst
		if !tt.dontConcatDst {
			dst = filepath.Join(tmpDir, tt.dst)
		}
		err := createDir(testTarget, dst, tt.name, tt.mode, tt.cfg)
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

		testTarget := target.NewOS()
		tmpDir := t.TempDir()
		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(testTarget, tmpDir)
		}
		dst := filepath.Join(tmpDir, tt.dst)
		err := createSymlink(testTarget, dst, tt.name, tt.linkTarget, tt.cfg)
		gotError := (err != nil)
		if tt.expectError != gotError {
			t.Errorf("[%v] createSymlink(dst=%s, name=%s, linkTarget=%s, denySymlinkExtraction=%v) = ERROR(%v); want %v", i, tt.dst, tt.name, tt.linkTarget, tt.cfg.DenySymlinkExtraction(), err, tt.expectError)
		}
	}
}

func TestSecurityCheck(t *testing.T) {
	tc := []struct {
		dst         string
		name        string
		cfg         *config.Config
		expectError bool
		prep        func(target.Target, string)
	}{
		{
			name: "test.txt",
			dst:  "",
		},
		{
			name: "",
			dst:  "",
		},
		{
			dst:  "foo",
			name: "bar",
		},
		{
			dst:  "foo",
			name: "bar/../baz",
		},
		{
			dst:         "foo",
			name:        "../baz",
			expectError: true,
		},
		{
			name: "foo/above/bar",
			prep: func(t target.Target, dst string) {
				if err := t.CreateDir(filepath.Join(dst, "foo"), 0750); err != nil {
					panic(fmt.Errorf("failed to create dir: %s", err))
				}

				above := filepath.Join(dst, "foo", "above")
				if err := t.CreateSymlink("../", above, false); err != nil {
					panic(fmt.Errorf("failed to create symlink: %s", err))
				}
			},
			expectError: true,
		},
	}

	for i, tt := range tc {
		testTarget := target.NewOS()
		tmp := t.TempDir()
		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(testTarget, tmp)
		}
		dst := filepath.Join(tmp, tt.dst)
		err := SecurityCheck(testTarget, dst, tt.name, tt.cfg)
		gotError := (err != nil)
		if tt.expectError != gotError {
			t.Errorf("[%v] securityCheck(dst=%s, name=%s) = ERROR(%v); want %v", i, tt.dst, tt.name, err, tt.expectError)
		}
	}
}

// FuzzSecurityCheckOs is a fuzzer for the SecurityCheck function
func FuzzSecurityCheckOs(f *testing.F) {
	f.Add("dst", "name")
	o := target.NewOS()
	f.Fuzz(func(t *testing.T, dst, name string) {
		tmp := t.TempDir()
		_ = SecurityCheck(o, tmp, name, config.NewConfig())
	})
}