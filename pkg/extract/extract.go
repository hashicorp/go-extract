package extract

import (
	"context"
	"os"
	"path/filepath"
)

func Extract(ctx context.Context, src, dst string) error {

	// Extractors
	var unzip Zip

	// TODO(jan): determine correct extractor

	// create tmp directory
	tmpDir, err := os.MkdirTemp(os.TempDir(), "extract*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	tmpDir = filepath.Clean(tmpDir) + string(os.PathSeparator)

	// TODO(jan): add timeout

	// extract zip
	if err := unzip.Extract(src, tmpDir); err != nil {
		return err
	}

	// move content from tmpDir to destination
	if err := CopyDirectory(tmpDir, dst); err != nil {
		return err
	}

	return nil
}

func writeSymbolicLink(filePath string, targetPath string) error {

	// create dirs
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	// create link
	if err := os.Symlink(targetPath, filePath); err != nil {
		return err
	}

	return nil
}
