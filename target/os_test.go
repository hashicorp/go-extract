package target

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewOS(t *testing.T) {
	os := NewOS()
	if os == nil {
		t.Errorf("NewOS() = %v, want non-nil", os)
	}
}

func TestCreateDir(t *testing.T) {

	name := "testdir"
	tmpDir := t.TempDir()
	os := NewOS()

	path := filepath.Join(tmpDir, name)
	err := os.CreateDir(path, 0755)
	if err != nil {
		t.Errorf("CreateDir() error = %v, want nil", err)
	}

	// create a test file
	fileName := "testfile"
	filePath := filepath.Join(path, fileName)
	data := "Hello, World!"
	reader := io.NopCloser(strings.NewReader(data))
	_, err = os.CreateFile(filePath, reader, 0644, false, -1)
	if err != nil {
		t.Errorf("CreateFile() error = %v, want nil", err)
	}

	// try create a directory below file and expect error
	subDirName := "subdir"
	subDirPath := filepath.Join(filePath, subDirName)
	err = os.CreateDir(subDirPath, 0755)
	if err == nil {
		t.Errorf("CreateDir() error = nil, want Error!")
	}

}

func TestCreateFile(t *testing.T) {
	tmpDir := t.TempDir()
	name := "testfile"
	path := filepath.Join(tmpDir, name)
	data := "Hello, World!"
	reader := io.NopCloser(strings.NewReader(data))

	os := NewOS()
	written, err := os.CreateFile(path, reader, 0644, false, -1)
	if err != nil {
		t.Errorf("CreateFile() error = %v, want nil", err)
	}

	if written != int64(len(data)) {
		t.Errorf("CreateFile() written = %v, want %v", written, len(data))
	}

	// try recreate, but expect error
	_, err = os.CreateFile(path, reader, 0644, false, -1)
	if err == nil {
		t.Errorf("CreateFile() error = nil, want Error!")
	}

	// try recreate, with overwrite
	_, err = os.CreateFile(path, reader, 0644, true, -1)
	if err != nil {
		t.Errorf("CreateFile() error = %v, want nil", err)
	}

	// try recreate, with overwrite and limit
	_, err = os.CreateFile(path, reader, 0644, true, 25)
	if err != nil {
		t.Errorf("CreateFile() error = %v, want nil", err)
	}

	// create a folder and try to overwrite with a file
	dirName := "testdir"
	dirPath := filepath.Join(tmpDir, dirName)
	err = os.CreateDir(dirPath, 0755)
	if err != nil {
		t.Errorf("CreateDir() error = %v, want nil", err)
	}
	_, err = os.CreateFile(dirPath, reader, 0644, true, -1)
	if err == nil {
		t.Errorf("CreateFile() error = nil, want Error!")
	}

	// create symlink and overwrite with a file
	linkName := "testlink"
	linkPath := filepath.Join(tmpDir, linkName)
	err = os.CreateSymlink(linkPath, "linktarget", false)
	if err != nil {
		t.Errorf("CreateSymlink() error = %v, want nil", err)
	}
	_, err = os.CreateFile(linkPath, reader, 0644, true, -1)
	if err != nil {
		t.Errorf("CreateFile() error = %v, want nil", err)
	}

}

func TestCreateSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	name := "testlink"
	target := "linktarget"
	path := filepath.Join(tmpDir, name)

	os := NewOS()
	err := os.CreateSymlink(path, target, false)
	if err != nil {
		t.Errorf("CreateSymlink() error = %v, want nil", err)
	}

	// try recreate, but expect error
	err = os.CreateSymlink(path, target, false)
	if err == nil {
		t.Errorf("CreateSymlink() error = nil, want Error!")
	}

	// try recreate, with overwrite
	err = os.CreateSymlink(path, target, true)
	if err != nil {
		t.Errorf("CreateSymlink() error = %v, want nil", err)
	}

	// create folder with file inside and try to overwrite folder with a symlink
	fileName := "testfile"
	reader := io.NopCloser(strings.NewReader("Hello, World!"))
	dirName := "testdir"
	dirPath := filepath.Join(tmpDir, dirName)
	err = os.CreateDir(dirPath, 0755)
	if err != nil {
		t.Errorf("CreateDir() error = %v, want nil", err)
	}
	_, err = os.CreateFile(filepath.Join(dirPath, fileName), reader, 0644, false, -1)
	if err != nil {
		t.Errorf("CreateFile() error = %v, want nil", err)
	}
	err = os.CreateSymlink(dirPath, target, true)
	if err == nil {
		t.Errorf("CreateSymlink() error = nil, want Error!")
	}

	// try to create a symlink to a non-existing directory
	err = os.CreateSymlink(filepath.Join(tmpDir, "nonexisting", "link"), target, false)
	if err == nil {
		t.Errorf("CreateSymlink() error = nil, want Error!")
	}

}

func TestLstat(t *testing.T) {
	tmpDir := t.TempDir()
	name := "testfile"
	path := filepath.Join(tmpDir, name)
	data := "Hello, World!"
	reader := io.NopCloser(strings.NewReader(data))

	os := NewOS()
	_, err := os.CreateFile(path, reader, 0644, false, -1)
	if err != nil {
		t.Errorf("CreateFile() error = %v, want nil", err)
	}
	_, err = os.Lstat(path)
	if err != nil {
		t.Errorf("Lstat() error = %v, want nil", err)
	}
}
