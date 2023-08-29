package target

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

func TestCreateSafeDir(t *testing.T) {

	testDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		t.Errorf(err.Error())
	}
	testDir = filepath.Clean(testDir) + string(os.PathSeparator)
	defer os.RemoveAll(testDir)
	syscall.Chdir(testDir)

	cases := []struct {
		name        string
		basePath    string
		newDir      string
		expectError bool
	}{
		{
			name:        "legit directory name",
			basePath:    ".",
			newDir:      "test",
			expectError: false,
		},
		{
			name:        "legit directory path",
			basePath:    ".",
			newDir:      "test/foo/bar",
			expectError: false,
		},
		{
			name:        "legit directory path with taversal",
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
			name:        "base with traversal, legit directory path with taversal",
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
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			target := &Os{}

			// perform actual test
			want := tc.expectError
			got := target.CreateSafeDir(config.NewConfig(), tc.basePath, tc.newDir) != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

func TestCreateSafeSymlink(t *testing.T) {

	testDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		t.Errorf(err.Error())
	}
	testDir = filepath.Clean(testDir) + string(os.PathSeparator)
	defer os.RemoveAll(testDir)
	syscall.Chdir(testDir)

	// test cases
	cases := []struct {
		name  string
		input struct {
			name   string
			target string
		}
		expectError bool
	}{
		{
			name: "legit link name",
			input: struct {
				name   string
				target string
			}{name: "foo", target: "bar"},
			expectError: false,
		},
		{
			name: "legit link in sub dir",
			input: struct {
				name   string
				target string
			}{name: "te/bar", target: "baz"},
			expectError: false,
		},
		{
			name: "legit link name with path with taversal",
			input: struct {
				name   string
				target string
			}{name: "test/../bar", target: "baz"},
			expectError: false,
		},
		{
			name: "malicious link name with path taversal",
			input: struct {
				name   string
				target string
			}{name: "../test", target: "baz"},
			expectError: true,
		},
		{
			name: "malicious link name with more complex path taversal",
			input: struct {
				name   string
				target string
			}{name: "./foo/bar/../test/../../../outside", target: "baz"},
			expectError: true,
		},
		{
			name: "legit link target",
			input: struct {
				name   string
				target string
			}{name: "test0", target: "foo"},
			expectError: false,
		},
		{
			name: "legit link target in subdir",
			input: struct {
				name   string
				target string
			}{name: "test1", target: "foo/bar"},
			expectError: false,
		},
		{
			name: "legit link target with path with taversal",
			input: struct {
				name   string
				target string
			}{name: "test2", target: "test/../bar"},
			expectError: false,
		},
		{
			name: "malicious link target with path taversal",
			input: struct {
				name   string
				target string
			}{name: "test3", target: "../baz"},
			expectError: true,
		},
		{
			name: "legit link",
			input: struct {
				name   string
				target string
			}{name: "foo/test3", target: "../baz"},
			expectError: false,
		},

		{
			name: "malicious link target with more complex path taversal",
			input: struct {
				name   string
				target string
			}{name: "test4", target: "./foo/bar/../test/../../../outside"},
			expectError: true,
		},
		{
			name: "malicious link target with absolut path",
			input: struct {
				name   string
				target string
			}{name: "test5", target: "/etc/passwd"},
			expectError: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				panic(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			target := &Os{}

			// perform actual tests
			want := tc.expectError

			err = nil
			err = target.CreateSafeSymlink(config.NewConfig(), testDir, tc.input.name, tc.input.target)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}
}

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
				mode:   0,
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
			config:      config.NewConfig(config.WithMaxFileSize(5)), // adjusted default
			expectError: true,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			// perform actual tests
			target := &Os{}
			want := tc.expectError
			err = target.CreateSafeFile(tc.config, testDir, tc.input.name, tc.input.reader, tc.input.mode)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}

}

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
			name: "normal behaviour does not allow overwrite",
			input: fnInput{
				name:   "foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0644,
			},
			config:      config.NewConfig(), // default settings are fine
			expectError: true,
		},
		{
			name: "allow overwrite",

			input: fnInput{
				name:   "aaa/bbb",
				reader: bytes.NewReader([]byte("data")),
				mode:   0644,
			},
			config:      config.NewConfig(config.WithForce(true)), // allow overwrite
			expectError: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

			// create testing directory
			testDir, err := os.MkdirTemp(os.TempDir(), "test*")
			if err != nil {
				t.Errorf(err.Error())
			}
			testDir = filepath.Clean(testDir) + string(os.PathSeparator)
			defer os.RemoveAll(testDir)

			// perform actual tests
			target := &Os{}
			want := tc.expectError
			// double extract
			err = target.CreateSafeFile(tc.config, testDir, tc.input.name, tc.input.reader, tc.input.mode)
			err = target.CreateSafeFile(tc.config, testDir, tc.input.name, tc.input.reader, tc.input.mode)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}

}

// func TestCheckForTraversal(t *testing.T) {

// 	cases := []struct {
// 		name        string
// 		basePath    string
// 		testPath    string
// 		expectError bool
// 	}{
// 		{
// 			name:        "legit path",
// 			basePath:    ".",
// 			testPath:    "test",
// 			expectError: false,
// 		},
// 		{
// 			name:        "legit path in sub",
// 			basePath:    ".",
// 			testPath:    "foo/test",
// 			expectError: false,
// 		},
// 		{
// 			name:        "legit path in other dir",
// 			basePath:    "/foo",
// 			testPath:    "bar",
// 			expectError: false,
// 		},
// 		{
// 			name:        "legit path with traversal",
// 			basePath:    "../",
// 			testPath:    "bar",
// 			expectError: false,
// 		},
// 		{
// 			name:        "legit base with traversal, malicious path",
// 			basePath:    "../",
// 			testPath:    "../bar",
// 			expectError: true,
// 		},

// 		{
// 			name:        "simple traversal",
// 			basePath:    ".",
// 			testPath:    "../test",
// 			expectError: true,
// 		},
// 	}

// 	// run cases
// 	for i, tc := range cases {
// 		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {

// 			// perform actual tests
// 			want := tc.expectError
// 			// double extract
// 			err := checkForTraversal(tc.basePath, tc.testPath)
// 			got := err != nil
// 			if got != want {
// 				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
// 			}

// 		})
// 	}

// }
