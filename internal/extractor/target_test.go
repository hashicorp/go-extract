package extractor

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

func testTargets(t *testing.T) []struct {
	name   string
	path   string
	link   string
	file   string
	data   []byte
	target Target
} {
	tmpDir := t.TempDir()
	testData := []byte("test data")
	return []struct {
		name   string
		path   string
		link   string
		file   string
		data   []byte
		target Target
	}{
		{
			name:   "os",
			path:   filepath.Join(tmpDir, "test"),
			link:   filepath.Join(tmpDir, "symlink"),
			file:   filepath.Join(tmpDir, "file"),
			data:   testData,
			target: NewOS(),
		},
		{
			name:   "Memory",
			path:   "test",
			link:   "symlink",
			file:   "file",
			data:   testData,
			target: NewMemory(),
		},
	}
}

// TestCreateSymlink tests the CreateSymlink function from Os
func TestCreateSymlink(t *testing.T) {

	for _, test := range testTargets(t) {
		t.Run(test.name, func(t *testing.T) {

			// create a file
			if _, err := test.target.CreateFile(test.path, bytes.NewReader(test.data), 0644, false, -1); err != nil {
				t.Fatalf("CreateFile() failed with an error, but no error was expected: %s", err)
			}

			// create a symlink
			if err := test.target.CreateSymlink(test.path, test.link, false); err != nil {
				t.Fatalf("CreateSymlink() failed with an error, but no error was expected: %s", err)
			}

			// check if symlink exists
			lstat, err := test.target.Lstat(test.link)
			if err != nil {
				t.Fatalf("Lstat() returned an error, but no error was expected: %s", err)
			}
			if lstat.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("CreateSymlink() failed: %s", "not a symlink")
			}

			// create a symlink with overwrite
			if err := test.target.CreateSymlink(test.link, test.path, true); err != nil {
				t.Fatalf("CreateSymlink() with overwrite failed, but no error was expected: %s", err)
			}

			// create a symlink with overwrite expect fail
			if err := test.target.CreateSymlink(test.link, test.path, false); err == nil {
				t.Fatalf("CreateSymlink() with disabled overwrite try to let the function fail, but error returned: %s", err)
			}

		})
	}
}

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
		prep        func(*testing.T, Target, string)
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
			prep: func(t *testing.T, target Target, dst string) {
				if err := target.CreateDir(filepath.Join(dst, "foo"), 0000); err != nil {
					t.Fatalf("failed to create dir: %s", err)
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

		var testTarget = NewOS()
		tmpDir := t.TempDir()

		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(t, testTarget, tmpDir)
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
		prep          func(*testing.T, Target, string)
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
			prep: func(t *testing.T, target Target, dst string) {
				if err := target.CreateDir(filepath.Join(dst, "foo"), 0000); err != nil {
					t.Fatalf("failed to create dir: %s", err)
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
			dst:         "",
			name:        "/failingt-extract",
			mode:        0750,
			expectError: runtime.GOOS != "windows", // bc, name is *not* concatenated with tmpDir.
			// The leading slash is not removed, but unimportant for windows
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

		testTarget := NewOS()
		tmpDir := t.TempDir()

		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(t, testTarget, tmpDir)
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

func TestSecurityCheck(t *testing.T) {
	tc := []struct {
		dst         string
		name        string
		cfg         *config.Config
		expectError bool
		prep        func(*testing.T, Target, string)
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
			prep: func(t *testing.T, target Target, dst string) {
				if err := target.CreateDir(filepath.Join(dst, "foo"), 0750); err != nil {
					t.Fatalf("failed to create dir: %s", err)
				}

				above := filepath.Join(dst, "foo", "above")
				if err := target.CreateSymlink("..", above, false); err != nil {
					t.Fatalf("failed to create symlink: %s", err)
				}
			},
			expectError: true,
		},
	}

	for i, tt := range tc {
		testTarget := NewOS()
		tmp := t.TempDir()
		if tt.cfg == nil {
			tt.cfg = config.NewConfig()
		}
		if tt.prep != nil {
			tt.prep(t, testTarget, tmp)
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
	o := NewOS()
	f.Fuzz(func(t *testing.T, dst, name string) {
		tmp := t.TempDir()
		_ = SecurityCheck(o, tmp, name, config.NewConfig())
	})
}
