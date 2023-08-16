package extract

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type extractor interface {
	Extract(e *Extract, src string, dst string) error
	FileSuffix() string
}

type Extract struct {
	// MaxFiles is the maximum of files in an archive
	MaxFiles int64

	// MaxFileSize is the maximum size of a file after decmpression
	MaxFileSize int64
}

// Create a new Extract object with defaults
func New() *Extract {
	return &Extract{
		MaxFiles:    1000,
		MaxFileSize: 1 << (10 * 3), // 1 Gb
	}
}

func (e *Extract) Unpack(ctx context.Context, src, dst string) error {

	var ex extractor

	if ex = e.findExtractor(src); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// create tmp directory
	tmpDir, err := os.MkdirTemp(os.TempDir(), "extract*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	tmpDir = filepath.Clean(tmpDir) + string(os.PathSeparator)

	// TODO(jan): add timeout

	// extract files in tmpDir
	if err := ex.Extract(e, src, tmpDir); err != nil {
		return err
	}

	// move content from tmpDir to destination
	if err := CopyDirectory(tmpDir, dst); err != nil {
		return err
	}

	return nil
}

// findExtractor identifies the correct extractor based on src filename with longest suffix match
func (e *Extract) findExtractor(src string) extractor {

	// TODO(jan): detect filetype based on magic bytes

	// Prepare available extractors
	extractors := []extractor{NewTar(), NewZip()}

	// find extractor with longest suffix match
	var maxSuffixLength int
	var engine extractor
	for _, ex := range extractors {

		// get suffix
		suff := ex.FileSuffix()

		// skip non-matching extractors
		if !strings.HasSuffix(src, suff) {
			continue
		}

		// check for longest suffix
		if len(suff) > maxSuffixLength {
			maxSuffixLength = len(suff)
			engine = ex
		}
	}

	return engine
}

// createDir creates in dstDir all directories that are provided in dirName
func (e *Extract) createDir(dstDir string, dirName string) error {

	// get absolut path
	tragetDir := filepath.Clean(filepath.Join(dstDir, dirName)) + string(os.PathSeparator)

	// check path
	if !strings.HasPrefix(tragetDir, dstDir) {
		return fmt.Errorf("filename path traversal detected: %v", dirName)
	}

	// create dirs
	if err := os.MkdirAll(tragetDir, os.ModePerm); err != nil {
		return err
	}

	return nil
}

// createSymlink creates in dstDir a symlink name with destination linkTarget
func (e *Extract) createSymlink(dstDir string, name string, linkTarget string) error {

	// create target dir
	if err := e.createDir(dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, name))

	// check absolut path
	// TODO(jan): check for windows
	// TODO(jan): network drives concideration on win `\\<remote>`
	if strings.HasPrefix(linkTarget, "/") {
		return fmt.Errorf("symlink absolut path detected: %v", linkTarget)
	}

	// check relative path
	canonicalTarget := filepath.Clean(filepath.Join(dstDir, linkTarget))
	if !strings.HasPrefix(canonicalTarget, dstDir) {
		return fmt.Errorf("symlink path traversal detected: %v", linkTarget)
	}

	// create link
	if err := os.Symlink(linkTarget, targetFilePath); err != nil {
		return err
	}

	return nil
}

// createFile creates name in dstDir with conte nt from reader and file
// headers as provided in mode
func (e *Extract) createFile(dstDir string, name string, reader io.Reader, mode fs.FileMode) error {

	// create target dir
	if err := e.createDir(dstDir, filepath.Dir(name)); err != nil {
		return err
	}

	// target file
	targetFilePath := filepath.Clean(filepath.Join(dstDir, name))

	// create dst file
	dstFile, err := os.OpenFile(targetFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() {
		dstFile.Close()
	}()

	// TODO(jan): filesize check
	if _, err := io.Copy(dstFile, reader); err != nil {
		return err
	}

	return nil
}

func (e *Extract) incrementAndCheckMaxFiles(counter *int64) error {
	*counter++
	if *counter > e.MaxFiles {
		return fmt.Errorf("to many files to extract")
	}
	return nil
}

func (e *Extract) checkFileSize(fileSize int64) error {
	if fileSize > e.MaxFileSize {
		return fmt.Errorf("maximum filesize exceeded")
	}
	return nil
}
