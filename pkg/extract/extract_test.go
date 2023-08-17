package extract

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestUnpack(t *testing.T) {
	panic("TODO!")
}

func TestFindExtractor(t *testing.T) {
	// test cases
	cases := []struct {
		name          string
		input         string
		extractEngine *Extract
		expected      extractor
	}{
		{
			name:          "get zip extractor from file",
			input:         "foo.zip",
			extractEngine: New(),
			expected:      NewZip(),
		},
		{
			name:          "get zip extractor from file in path",
			input:         "foo.zip",
			extractEngine: New(),
			expected:      NewZip(),
		},
		{
			name:          "get tar extractor from file",
			input:         "foo.tar",
			extractEngine: New(),
			expected:      NewTar(),
		},
		{
			name:          "get tar extractor from file in path",
			input:         "foo.tar",
			extractEngine: New(),
			expected:      NewTar(),
		},
		{
			name:          "unspported file type .7z",
			input:         "foo.7z",
			extractEngine: New(),
			expected:      nil,
		},
		{
			name:          "no filetype",
			input:         "foo",
			extractEngine: New(),
			expected:      nil,
		},
		{
			name:          "camel case",
			input:         "foo.zIp",
			extractEngine: New(),
			expected:      NewZip(),
		},
		{
			name:          "camel case",
			input:         "foo.TaR",
			extractEngine: New(),
			expected:      NewTar(),
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			// prepare vars
			var failed bool
			want := tc.expected

			// perform actual tests
			got := tc.extractEngine.findExtractor(tc.input)

			// success if both are nil and no engine found
			if want == got {
				return
			}

			// check if engine detection failed
			if got == nil {
				failed = true
			}

			// if not failed yet, compare identified suffixes
			if !failed {
				if got.FileSuffix() != want.FileSuffix() {
					failed = true
				}
			}

			if failed {
				t.Errorf("test case %d failed: %s\nexpected: %v\ngot: %v", i, tc.name, want, got)
			}

		})
	}

}

func TestCreateDir(t *testing.T) {

	// create testing directory
	testDir, err := os.MkdirTemp(os.TempDir(), "test*")
	if err != nil {
		t.Errorf(err.Error())
	}
	defer os.RemoveAll(testDir)
	testDir = filepath.Clean(testDir) + string(os.PathSeparator)

	cases := []struct {
		name          string
		input         string
		extractEngine *Extract
		expectError   bool
	}{
		{
			name:          "legit directory name",
			input:         "test",
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name:          "legit directory path",
			input:         "test/foo/bar",
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name:          "legit directory path with taversal",
			input:         "test/foo/../bar",
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name:          "just the current dir",
			input:         ".",
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name:          "directory traversal",
			input:         "../foo",
			extractEngine: New(), // default settings are fine
			expectError:   true,
		},
		{
			name:          "more tricky traversal",
			input:         "./test/../foo/../../outside",
			extractEngine: New(), // default settings are fine
			expectError:   true,
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

			// perform actual test
			want := tc.expectError
			got := tc.extractEngine.createDir(testDir, tc.input) != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

func TestCreateSymlink(t *testing.T) {

	// test cases
	cases := []struct {
		name  string
		input struct {
			name   string
			target string
		}
		extractEngine *Extract
		expectError   bool
	}{
		{
			name: "legit link name",
			input: struct {
				name   string
				target string
			}{name: "foo", target: "bar"},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "legit link in sub dir",
			input: struct {
				name   string
				target string
			}{name: "test/foo/bar", target: "baz"},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "legit link name with path with taversal",
			input: struct {
				name   string
				target string
			}{name: "test/../bar", target: "baz"},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "malicious link name with path taversal",
			input: struct {
				name   string
				target string
			}{name: "../test", target: "baz"},
			extractEngine: New(), // default settings are fine
			expectError:   true,
		},
		{
			name: "malicious link name with more complex path taversal",
			input: struct {
				name   string
				target string
			}{name: "./foo/bar/../test/../../../outside", target: "baz"},
			extractEngine: New(), // default settings are fine
			expectError:   true,
		},
		{
			name: "legit link target",
			input: struct {
				name   string
				target string
			}{name: "test0", target: "foo"},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "legit link target in subdir",
			input: struct {
				name   string
				target string
			}{name: "test1", target: "foo/bar"},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "legit link target with path with taversal",
			input: struct {
				name   string
				target string
			}{name: "test2", target: "test/../bar"},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "malicious link target with path taversal",
			input: struct {
				name   string
				target string
			}{name: "test3", target: "../baz"},
			extractEngine: New(), // default settings are fine
			expectError:   true,
		},
		{
			name: "malicious link target with more complex path taversal",
			input: struct {
				name   string
				target string
			}{name: "test4", target: "./foo/bar/../test/../../../outside"},
			extractEngine: New(), // default settings are fine
			expectError:   true,
		},
		{
			name: "malicious link target with absolut path",
			input: struct {
				name   string
				target string
			}{name: "test5", target: "/etc/passwd"},
			extractEngine: New(), // default settings are fine
			expectError:   true,
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
			want := tc.expectError
			err = tc.extractEngine.createSymlink(testDir, tc.input.name, tc.input.target)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}
}

func TestCreateFile(t *testing.T) {

	// test cases

	type fnInput struct {
		name   string
		reader io.Reader
		mode   fs.FileMode
	}

	cases := []struct {
		name          string
		input         fnInput
		extractEngine *Extract
		expectError   bool
	}{
		{
			name: "legit file",
			input: fnInput{
				name:   "foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "legit file in sub-dir",
			input: fnInput{
				name:   "test/foo",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "legit file in sub-dir with legit traversal",
			input: fnInput{
				name:   "test/foo/../bar",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			extractEngine: New(), // default settings are fine
			expectError:   false,
		},
		{
			name: "malicious file with traversal",
			input: fnInput{
				name:   "../bar",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			extractEngine: New(), // default settings are fine
			expectError:   true,
		},
		{
			name: "malicious file with traversal, more complex",
			input: fnInput{
				name:   "./test/../bar/../foo/../../../../../../../../../tmp/test",
				reader: bytes.NewReader([]byte("data")),
				mode:   0,
			},
			extractEngine: New(), // default settings are fine
			expectError:   true,
		},
		{
			name: "malicious file with too much content",
			input: fnInput{
				name:   "test",
				reader: bytes.NewReader([]byte("1234567890")), // 10 byte file content
				mode:   0,
			},
			extractEngine: &Extract{MaxFileSize: 5}, // allow only 5 byte files
			expectError:   true,
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
			want := tc.expectError
			err = tc.extractEngine.createFile(testDir, tc.input.name, tc.input.reader, tc.input.mode)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s\n%v", i, tc.name, err)
			}

		})
	}

	// dstDir string, name string, reader io.Reader, mode fs.FileMode

}

func TestCheckMaxFiles(t *testing.T) {
	// prepare testcases
	cases := []struct {
		name          string
		input         int64
		extractEngine Extract
		expectError   bool
	}{
		{
			name:          "less files then maximum",
			input:         5,                     // within limit
			extractEngine: Extract{MaxFiles: 10}, // 10
			expectError:   false,
		},
		{
			name:          "more files then maximum",
			input:         15,                    // over limit
			extractEngine: Extract{MaxFiles: 10}, // 10
			expectError:   true,
		},
		{
			name:          "disable file counter check",
			input:         5000,                  // ignored
			extractEngine: Extract{MaxFiles: -1}, // disable
			expectError:   false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expectError
			got := tc.extractEngine.checkMaxFiles(tc.input) != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}

func TestCheckFileSize(t *testing.T) {

	// prepare testcases
	cases := []struct {
		name          string
		input         int64
		extractEngine Extract
		expectError   bool
	}{
		{
			name:          "file size less then maximum",
			input:         1 << (9 * 1),                        // 512b
			extractEngine: Extract{MaxFileSize: 1 << (10 * 1)}, // 1kb
			expectError:   false,
		},
		{
			name:          "file bigger then maximum",
			input:         5 << (10 * 1),                       // 5kb
			extractEngine: Extract{MaxFileSize: 1 << (10 * 1)}, // 1 kb
			expectError:   true,
		},
		{
			name:          "disable filzes check",
			input:         5 << (10 * 1),            // 5kb
			extractEngine: Extract{MaxFileSize: -1}, // disable
			expectError:   false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			want := tc.expectError
			got := tc.extractEngine.checkFileSize(tc.input) != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}
}
