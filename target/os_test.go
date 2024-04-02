package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

// TestCreateSafeDir implements test cases
func TestCreateSafeDir(t *testing.T) {

	cases := []struct {
		name        string
		basePath    string
		newDir      string
		cfg         *config.Config
		expectError bool
	}{
		{
			name:        "legit directory name",
			basePath:    ".",
			newDir:      "test",
			expectError: false,
		},
		{
			name:        "legit directory name, remove start of absolute path",
			basePath:    ".",
			newDir:      "/test",
			expectError: false,
		},
		{
			name:        "legit directory path",
			basePath:    ".",
			newDir:      "test/foo/bar",
			expectError: false,
		},
		{
			name:        "legit directory path with traversal",
			basePath:    ".",
			newDir:      "test/foo/../bar",
			expectError: false,
		},
		{
			name:        "just the current dir",
			basePath:    ".",
			newDir:      ".",
			expectError: false,
		},
		{
			name:        "directory traversal",
			basePath:    ".",
			newDir:      "../foo",
			expectError: true,
		},
		{
			name:        "non-existent base-dir",
			basePath:    "foo",
			newDir:      "bar",
			expectError: true,
		},
		{
			name:        "create sub-dir in non-existent base-dir",
			basePath:    "foo",
			newDir:      "bar",
			cfg:         config.NewConfig(config.WithCreateDestination(true)),
			expectError: false,
		},
		{
			name:        "create sub-dir in non-existent base-dir including traversal",
			basePath:    "../foo",
			newDir:      "bar",
			cfg:         config.NewConfig(config.WithCreateDestination(true)),
			expectError: false,
		},
		{
			name:        "more tricky traversal",
			basePath:    ".",
			newDir:      "./test/../foo/../../outside",
			expectError: true,
		},

		{
			name:        "base with traversal, legit directory name",
			basePath:    "..",
			newDir:      "test",
			expectError: false,
		},
		{
			name:        "base with traversal, legit directory path",
			basePath:    "..",
			newDir:      "test/foo/bar",
			expectError: false,
		},
		{
			name:        "base with traversal, legit directory path with traversal",
			basePath:    "..",
			newDir:      "test/foo/../bar",
			expectError: false,
		},
		{
			name:        "base with traversal, just the current dir",
			basePath:    "..",
			newDir:      ".",
			expectError: false,
		},
		{
			name:        "base with traversal, directory traversal",
			basePath:    "..",
			newDir:      "../foo",
			expectError: true,
		},
		{
			name:        "base with traversal, more tricky traversal",
			basePath:    "..",
			newDir:      "./test/../foo/../../outside",
			expectError: true,
		},
		{
			name:        "absolute path and traversal",
			basePath:    "/tmp/foo",
			newDir:      "./test/../foo/../../outside",
			expectError: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			// create testing directory
			testDir := t.TempDir()

			// create a sub dir for path traversal testing
			testDir = filepath.Join(testDir, "base")
			if err := os.Mkdir(testDir, os.ModePerm); err != nil {
				t.Errorf(err.Error())
			}

			target := &OS{}

			// check config
			var cfg *config.Config
			if tc.cfg == nil {
				cfg = config.NewConfig()
			} else {
				cfg = tc.cfg
			}

			// perform actual test
			want := tc.expectError
			err := target.CreateSafeDir(cfg, filepath.Join(testDir, tc.basePath), tc.newDir, fs.FileMode(cfg.DefaultDirPermission()))
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}
		})
	}
}

func TestCreateSafeSymlink(t *testing.T) {

	type fnInput struct {
		name   string
		target string
	}

	// test cases
	cases := []struct {
		name        string
		input       fnInput
		cfg         *config.Config
		expectError bool
	}{
		{
			name:        "legit link name",
			input:       fnInput{name: "foo", target: "bar"},
			expectError: false,
		},
		{
			name:        "legit link name",
			input:       fnInput{name: "foo", target: "bar"},
			cfg:         config.NewConfig(config.WithDenySymlinkExtraction(true)),
			expectError: false,
		},
		{
			name:        "legit link in sub dir",
			input:       fnInput{name: "te/bar", target: "baz"},
			expectError: false,
		},
		{
			name:        "legit link name with path with traversal",
			input:       fnInput{name: "test/../bar", target: "baz"},
			expectError: false,
		},
		{
			name:        "malicious link name with path traversal",
			input:       fnInput{name: "../test", target: "baz"},
			expectError: true,
		},
		{
			name:        "malicious link name with more complex path traversal",
			input:       fnInput{name: "./foo/bar/../test/../../../outside", target: "baz"},
			expectError: true,
		},
		{
			name:        "legit link target",
			input:       fnInput{name: "test0", target: "foo"},
			expectError: false,
		},
		{
			name:        "legit link target in sub-dir",
			input:       fnInput{name: "test1", target: "foo/bar"},
			expectError: false,
		},
		{
			name:        "legit link target with path with traversal",
			input:       fnInput{name: "test2", target: "test/../bar"},
			expectError: false,
		},
		{
			name:        "malicious link target with path traversal",
			input:       fnInput{name: "test3", target: "../baz"},
			expectError: true,
		},
		{
			name:        "legit link",
			input:       fnInput{name: "foo/test3", target: "../baz"},
			expectError: false,
		},

		{
			name:        "malicious link target with more complex path traversal",
			input:       fnInput{name: "test4", target: "./foo/bar/../test/../../../outside"},
			expectError: true,
		},
		{
			name:        "malicious link target with absolute path linux",
			input:       fnInput{name: "test5", target: "/etc/passwd"},
			expectError: true,
		},
		{
			name:        "malicious link target with absolute path windows",
			input:       fnInput{name: "test6", target: "C:\\windows\\Systems32"},
			expectError: true,
		},
		{
			name:        "malicious link target with absolute path windows, but continue on error",
			input:       fnInput{name: "test6", target: "C:\\windows\\Systems32"},
			cfg:         config.NewConfig(config.WithContinueOnError(true)),
			expectError: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			// create testing directory
			testDir := t.TempDir()

			target := &OS{}
			cfg := config.NewConfig()
			if tc.cfg != nil {
				cfg = tc.cfg
			}

			// perform actual tests
			want := tc.expectError
			err := target.CreateSafeSymlink(cfg, testDir, tc.input.name, tc.input.target)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}
}

func TestCreateSafeSymlink_overwriteTest(t *testing.T) {

	// test creation of two symlinks
	// create testing directory
	testDir := t.TempDir()
	target := &OS{}
	if err := target.CreateSafeSymlink(config.NewConfig(), testDir, "foo", "bar"); err != nil {
		t.Errorf(err.Error())
	}

	type fnInput struct {
		name   string
		target string
	}

	// test cases
	cases := []struct {
		name        string
		input       fnInput
		cfg         *config.Config
		expectError bool
	}{
		{
			name:        "existing symlink overwritten",
			input:       fnInput{name: "foo", target: "baz"},
			cfg:         config.NewConfig(config.WithOverwrite(true)),
			expectError: false,
		},
		{
			name:        "existing symlink overwritten, but not configured",
			input:       fnInput{name: "foo", target: "baz"},
			cfg:         config.NewConfig(),
			expectError: true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expectError
			err := target.CreateSafeSymlink(tc.cfg, testDir, tc.input.name, tc.input.target)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}
		})
	}
}

// TestCreateSafeFile implements test cases
func TestCreateSafeFile(t *testing.T) {

	// test cases

	type fnInput struct {
		name   string
		reader io.Reader
		mode   fs.FileMode
	}

	cases := []struct {
		name        string
		input       fnInput
		config      *config.Config
		expectError bool
	}{
		{
			name: "legit file",
			input: fnInput{
				name:   "foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0640,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: false,
		},
		{
			name: "legit file",
			input: fnInput{
				name:   "foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0640,
			},
			config:      config.NewConfig(config.WithMaxExtractionSize(-1)), // Extraction without limit of dst size
			expectError: false,
		},
		{
			name: "legit file, without name",
			input: fnInput{
				name:   "",
				reader: bytes.NewReader([]byte("data")),
				mode:   0640,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: true,
		},
		{
			name: "remove absolute path prefix from file",
			input: fnInput{
				name:   "/foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0640,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: false,
		},

		{
			name: "legit file in sub-dir",
			input: fnInput{
				name:   "test/foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: false,
		},
		{
			name: "legit file in sub-dir with legit traversal",
			input: fnInput{
				name:   "test/foo/../bar",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: false,
		},
		{
			name: "malicious file with traversal",
			input: fnInput{
				name:   "../bar",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: true,
		},
		{
			name: "malicious file with traversal, more complex",
			input: fnInput{
				name:   "./test/../bar/../foo/../../../../../../../../../tmp/test",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: true,
		},
		{
			name: "malicious file with too much content",
			input: fnInput{
				name:   "test",
				reader: bytes.NewReader([]byte("1234567890")), // 10 byte file content
				mode:   0,
			},
			config:      config.NewConfig(config.WithMaxExtractionSize(5)), // adjusted default
			expectError: true,
		},
	}

	dir, _ := os.Getwd()
	log.Printf("testing-base-dir: %s", dir)

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			dir, _ := os.Getwd()
			log.Printf("test-start-dir: %s", dir)
			// create testing directory
			testDir := t.TempDir()

			// perform actual tests
			target := &OS{}
			want := tc.expectError
			err := target.CreateSafeFile(tc.config, testDir, tc.input.name, tc.input.reader, tc.input.mode)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s", i, tc.name, err)
			}

		})
	}

}

// TestOverwriteFile implements a test case
func TestOverwriteFile(t *testing.T) {

	// test cases

	type fnInput struct {
		name   string
		reader io.Reader
		mode   fs.FileMode
	}

	cases := []struct {
		name        string
		input       fnInput
		config      *config.Config
		expectError bool
	}{
		{
			name: "normal behaviors does not allow overwrite",
			input: fnInput{
				name:   "foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0640,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: true,
		},
		{
			name: "allow overwrite",

			input: fnInput{
				name:   "aaa/bbb",
				reader: bytes.NewReader([]byte("data")),
				mode:   0640,
			},
			config:      config.NewConfig(config.WithOverwrite(true)), // allow overwrite
			expectError: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			// create testing directory
			testDir := t.TempDir()

			// perform actual tests
			target := &OS{}
			want := tc.expectError
			// double extract
			err1 := target.CreateSafeFile(tc.config, testDir, tc.input.name, tc.input.reader, tc.input.mode)
			err2 := target.CreateSafeFile(tc.config, testDir, tc.input.name, tc.input.reader, tc.input.mode)
			got := err1 != nil || err2 != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%s\n%s", i, tc.name, err1, err2)
			}

		})
	}

}

func TestIsSymlink(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "temp")
	if err != nil {
		t.Fatal(err)
	}
	tempFile.Close()

	// Create a symlink to the temporary file
	symlinkPath := tempFile.Name() + ".symlink"
	err = os.Symlink(tempFile.Name(), symlinkPath)
	if err != nil {
		t.Fatal(err)
	}

	// Remember to clean up afterwards
	defer os.Remove(tempFile.Name())
	defer os.Remove(symlinkPath)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Symlink path",
			path: symlinkPath,
			want: true,
		},
		{
			name: "Non-symlink path",
			path: tempFile.Name(),
			want: false,
		},
		{
			name: "Empty path",
			path: "",
			want: false,
		},
		{
			name: "Current directory",
			path: ".",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSymlink(tt.path); got != tt.want {
				t.Errorf("isSymlink() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecurityCheckPath(t *testing.T) {

	cases := []struct {
		name        string
		basePath    string
		newDir      string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "legit directory name",
			basePath:    ".",
			newDir:      "test",
			config:      config.NewConfig(),
			expectError: false,
		},
		{
			name:        "traversal",
			basePath:    ".",
			newDir:      "../test",
			config:      config.NewConfig(),
			expectError: true,
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// perform actual test
			want := tc.expectError
			err := securityCheckPath(tc.config, tc.basePath, tc.newDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}

	// test with symlinks
	// create testing directory with symlink to current dir
	testDir := t.TempDir()
	symlink := filepath.Join(testDir, "symlink")
	if err := os.Symlink(".", symlink); err != nil {
		t.Errorf(err.Error())
	}

	// perform actual test
	cases = []struct {
		name        string
		basePath    string
		newDir      string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "deny follow symlink",
			newDir:      filepath.Join("symlink", "deny"),
			config:      config.NewConfig(),
			expectError: true,
		},
		{
			name:        "allow follow symlink",
			newDir:      filepath.Join("symlink", "allow"),
			config:      config.NewConfig(config.WithFollowSymlinks(true)),
			expectError: false,
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.expectError
			err := securityCheckPath(tc.config, testDir, tc.newDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s (%v)", i, tc.name, err)
			}
		})
	}

}

func TestGetStartOfAbsolutePath(t *testing.T) {
	cases := []struct {
		path string
	}{
		{
			path: "test",
		}, {
			path: "/test",
		}, {
			path: "//test",
		}, {
			path: "/c:\\/test",
		}, {
			path: "/c:\\/d:\\test",
		}, {
			path: "a:\\/c:\\/test",
		}, {
			path: `\\test`,
		},
	}

	// perform tests and expect always "test" as a result
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			if start := GetStartOfAbsolutePath(tc.path); strings.TrimPrefix(tc.path, start) != "test" {
				t.Errorf("test case %d failed: %s != test", i, strings.TrimPrefix(tc.path, start))
			}
		})
	}
}
