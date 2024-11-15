package extract_test

import (
	"testing"

	extract "github.com/hashicorp/go-extract"
)

// func testTargets(t *testing.T) []struct {
// 	name   string
// 	path   string
// 	link   string
// 	file   string
// 	data   []byte
// 	target extract.Target
// } {
// 	tmpDir := t.TempDir()
// 	testData := []byte("test data")
// 	return []struct {
// 		name   string
// 		path   string
// 		link   string
// 		file   string
// 		data   []byte
// 		target extract.Target
// 	}{
// 		{
// 			name:   "os",
// 			path:   filepath.Join(tmpDir, "test"),
// 			link:   filepath.Join(tmpDir, "symlink"),
// 			file:   filepath.Join(tmpDir, "file"),
// 			data:   testData,
// 			target: extract.NewDisk(),
// 		},
// 		{
// 			name:   "Memory",
// 			path:   "test",
// 			link:   "symlink",
// 			file:   "file",
// 			data:   testData,
// 			target: extract.NewMemory(),
// 		},
// 	}
// }

// func TestCreateSymlink(t *testing.T) {
// 	for _, test := range testTargets(t) {
// 		t.Run(test.name, func(t *testing.T) {
// 			// create a file
// 			if _, err := test.target.CreateFile(test.path, bytes.NewReader(test.data), 0644, false, -1); err != nil {
// 				t.Fatalf("CreateFile() failed with an error, but no error was expected: %s", err)
// 			}

// 			// create a symlink
// 			if err := test.target.CreateSymlink(test.path, test.link, false); err != nil {
// 				t.Fatalf("CreateSymlink() failed with an error, but no error was expected: %s", err)
// 			}

// 			// check if symlink exists
// 			lstat, err := test.target.Lstat(test.link)
// 			if err != nil {
// 				t.Fatalf("Lstat() returned an error, but no error was expected: %s", err)
// 			}
// 			if lstat.Mode()&os.ModeSymlink == 0 {
// 				t.Fatalf("CreateSymlink() failed: %s", "not a symlink")
// 			}

// 			// create a symlink with overwrite
// 			if err := test.target.CreateSymlink(test.link, test.path, true); err != nil {
// 				t.Fatalf("CreateSymlink() with overwrite failed, but no error was expected: %s", err)
// 			}

// 			// create a symlink with overwrite expect fail
// 			if err := test.target.CreateSymlink(test.link, test.path, false); err == nil {
// 				t.Fatalf("CreateSymlink() with disabled overwrite try to let the function fail, but error returned: %s", err)
// 			}

// 		})
// 	}
// }

// func TestCreateFile(t *testing.T) {
// 	tests := []struct {
// 		dst         string
// 		name        string
// 		src         string
// 		mode        fs.FileMode
// 		maxSize     int64
// 		cfg         *extract.Config
// 		expectError bool
// 		prep        func(*testing.T, extract.Target, string)
// 	}{
// 		{
// 			name:    "test.txt",
// 			src:     "Hello, World!",
// 			mode:    0640,
// 			maxSize: -1,
// 		},
// 		{
// 			name:        "",
// 			src:         "Hello, World!",
// 			mode:        0640,
// 			maxSize:     -1,
// 			expectError: true,
// 		},
// 		{
// 			dst:     "test",
// 			name:    "test.txt",
// 			src:     "Hello, World!",
// 			mode:    0640,
// 			maxSize: -1,
// 			cfg:     extract.NewConfig(extract.WithCreateDestination(true)),
// 		},
// 		{
// 			dst:     "foo/bar",
// 			name:    "test.txt",
// 			src:     "Hello, World!",
// 			mode:    0640,
// 			maxSize: -1,
// 			cfg:     extract.NewConfig(extract.WithCreateDestination(true)),
// 			prep: func(t *testing.T, target extract.Target, dst string) {
// 				if err := target.CreateDir(filepath.Join(dst, "foo"), 0000); err != nil {
// 					t.Fatalf("failed to create dir: %s", err)
// 				}
// 			},
// 			expectError: runtime.GOOS != "windows", // only relevant test for unix based systems
// 		},
// 		{
// 			dst:         "foo",
// 			name:        "test.txt",
// 			src:         "Hello, World!",
// 			mode:        0640,
// 			maxSize:     -1,
// 			cfg:         extract.NewConfig(extract.WithCreateDestination(false)),
// 			expectError: true,
// 		},
// 	}

// 	for _, test := range tests {
// 		var testTarget = extract.NewDisk()
// 		tmpDir := t.TempDir()

// 		if test.cfg == nil {
// 			test.cfg = extract.NewConfig()
// 		}
// 		if test.prep != nil {
// 			test.prep(t, testTarget, tmpDir)
// 		}
// 		dst := filepath.Join(tmpDir, test.dst)
// 		_, err := extract.createFile(testTarget, dst, test.name, strings.NewReader(test.src), test.mode, test.maxSize, test.cfg)
// 		if test.expectError != (err != nil) {
// 			t.Errorf("createFile(%s, %s, %s, %d, %d) = %v; want %v", test.dst, test.name, test.src, test.mode, test.maxSize, err, test.expectError)
// 		}
// 	}
// }

// func TestCreateDir(t *testing.T) {
// 	tests := []struct {
// 		dst           string
// 		name          string
// 		mode          fs.FileMode
// 		cfg           *extract.Config
// 		expectError   bool
// 		prep          func(*testing.T, extract.Target, string)
// 		dontConcatDst bool
// 	}{
// 		{
// 			name: "test",
// 			mode: 0750,
// 		},
// 		{
// 			name:        "",
// 			mode:        0750,
// 			expectError: false,
// 		},
// 		{
// 			dst:  "foo",
// 			name: "bar",
// 			mode: 0750,
// 			cfg:  extract.NewConfig(extract.WithCreateDestination(true)),
// 		},
// 		{
// 			dst:         "foo",
// 			name:        "bar",
// 			mode:        0750,
// 			cfg:         extract.NewConfig(extract.WithCreateDestination(false)),
// 			expectError: true,
// 		},
// 		{
// 			dst:  "foo",
// 			name: "bar",
// 			mode: 0750,
// 			cfg:  extract.NewConfig(extract.WithCreateDestination(true)),
// 			prep: func(t *testing.T, target extract.Target, dst string) {
// 				if err := target.CreateDir(filepath.Join(dst, "foo"), 0000); err != nil {
// 					t.Fatalf("failed to create dir: %s", err)
// 				}
// 			},
// 			expectError: (runtime.GOOS != "windows"), // only relevant test for unix based systems
// 		},
// 		{
// 			dst:         "",
// 			name:        "/failingt-extract",
// 			mode:        0750,
// 			expectError: false, // bc, name is concatenated with tmpDir
// 		},
// 		{
// 			dst:         "",
// 			name:        "/failingt-extract",
// 			mode:        0750,
// 			expectError: runtime.GOOS != "windows", // bc, name is *not* concatenated with tmpDir.
// 			// The leading slash is not removed, but unimportant for windows
// 			dontConcatDst: true,
// 		},
// 		{
// 			dst:           "",
// 			name:          "./failingt-extract",
// 			mode:          0750,
// 			expectError:   false,
// 			dontConcatDst: true,
// 		},
// 	}

// 	for _, test := range tests {
// 		testTarget := extract.NewDisk()
// 		tmpDir := t.TempDir()

// 		if test.cfg == nil {
// 			test.cfg = extract.NewConfig()
// 		}
// 		if test.prep != nil {
// 			test.prep(t, testTarget, tmpDir)
// 		}
// 		dst := test.dst
// 		if !test.dontConcatDst {
// 			dst = filepath.Join(tmpDir, test.dst)
// 		}
// 		err := createDir(testTarget, dst, test.name, test.mode, test.cfg)
// 		gotError := (err != nil)
// 		if test.expectError != gotError {
// 			t.Errorf("reateDir(dst=%s, name=%s, mode=%o, createDest=%v, defaultDirPerm=%o) = ERROR(%v); want %v", test.dst, test.name, test.mode.Perm(), test.cfg.CreateDestination(), test.cfg.CustomCreateDirMode(), err, test.expectError)
// 		}
// 	}
// }

// func TestSecurityCheck(t *testing.T) {
// 	tests := []struct {
// 		dst         string
// 		name        string
// 		cfg         *extract.Config
// 		expectError bool
// 		prep        func(*testing.T, extract.Target, string)
// 	}{
// 		{
// 			name: "test.txt",
// 			dst:  "",
// 		},
// 		{
// 			name: "",
// 			dst:  "",
// 		},
// 		{
// 			dst:  "foo",
// 			name: "bar",
// 		},
// 		{
// 			dst:  "foo",
// 			name: "bar/../baz",
// 		},
// 		{
// 			dst:         "foo",
// 			name:        "../baz",
// 			expectError: true,
// 		},
// 		{
// 			name: "foo/above/bar",
// 			prep: func(t *testing.T, target extract.Target, dst string) {
// 				if err := target.CreateDir(filepath.Join(dst, "foo"), 0750); err != nil {
// 					t.Fatalf("failed to create dir: %s", err)
// 				}

// 				above := filepath.Join(dst, "foo", "above")
// 				if err := target.CreateSymlink("..", above, false); err != nil {
// 					t.Fatalf("failed to create symlink: %s", err)
// 				}
// 			},
// 			expectError: true,
// 		},
// 	}

// 	for _, test := range tests {
// 		testTarget := extract.NewDisk()
// 		tmp := t.TempDir()
// 		if test.cfg == nil {
// 			test.cfg = extract.NewConfig()
// 		}
// 		if test.prep != nil {
// 			test.prep(t, testTarget, tmp)
// 		}
// 		dst := filepath.Join(tmp, test.dst)
// 		err := extract.SecurityCheck(testTarget, dst, test.name, test.cfg)
// 		gotError := (err != nil)
// 		if test.expectError != gotError {
// 			t.Errorf("securityCheck(dst=%s, name=%s) = ERROR(%v); want %v", test.dst, test.name, err, test.expectError)
// 		}
// 	}
// }

// FuzzSecurityCheckDisk is a fuzzer for the SecurityCheck function
func FuzzSecurityCheckDisk(f *testing.F) {
	f.Add("dst", "name")
	d := extract.NewDisk()
	f.Fuzz(func(t *testing.T, dst, name string) {
		tmp := t.TempDir()
		_ = extract.SecurityCheck(d, tmp, name, extract.NewConfig())
	})
}
