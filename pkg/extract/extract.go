package extract

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type extractor interface {
	Extract(e *Extract, src string, dst string) error
	FileSuffix() string
}

type Extract struct {
	// MaxFiles is the maximum of files in an archive.
	// Set value to -1 to disable the check.
	MaxFiles int64

	// MaxFileSize is the maximum size of a file after decompression.
	// Set value to -1 to disable the check.
	MaxFileSize int64

	// Maximum time in seconds that an extraction should need to finish
	MaxExtractionTime int64
}

// Create a new Extract object with defaults
func New() *Extract {
	return &Extract{
		MaxFiles:          1000,
		MaxFileSize:       1 << (10 * 3), // 1 Gb
		MaxExtractionTime: 60,            // 1 minute
	}
}

func (e *Extract) Unpack(ctx context.Context, src, dst string) error {

	// identify extraction engine
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

	// check if extraction timeout is set
	if e.MaxExtractionTime == -1 {
		if err := ex.Extract(e, src, dst); err != nil {
			return err
		}
	} else {
		if err := e.extractWithTimeout(ex, src, tmpDir); err != nil {
			return err
		}
	}

	// move content from tmpDir to destination
	if err := CopyDirectory(tmpDir, dst); err != nil {
		return err
	}

	return nil
}

func (e *Extract) extractWithTimeout(ex extractor, src string, dst string) error {
	// prepare extraction process
	exChan := make(chan error, 1)
	go func() {
		// extract files in tmpDir
		if err := ex.Extract(e, src, dst); err != nil {
			exChan <- err
		}
		exChan <- nil
	}()

	// start extraction in on thread
	select {
	case err := <-exChan:
		if err != nil {
			return err
		}
	case <-time.After(time.Duration(e.MaxExtractionTime) * time.Second):
		return fmt.Errorf("maximum extraction time exceeded")
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

	// finaly copy the data
	if _, err := io.Copy(dstFile, reader); err != nil {
		return err
	}

	return nil
}

func (e *Extract) incrementAndCheckMaxFiles(counter *int64) error {
	*counter++

	// check if disabled
	if e.MaxFiles == -1 {
		return nil
	}

	// check value
	if *counter > e.MaxFiles {
		return fmt.Errorf("to many files to extract")
	}
	return nil
}

func (e *Extract) checkFileSize(fileSize int64) error {

	// check if disabled
	if e.MaxFileSize == -1 {
		return nil
	}

	// check value
	if fileSize > e.MaxFileSize {
		return fmt.Errorf("maximum filesize exceeded")
	}
	return nil
}
