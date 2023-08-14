package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func DetermineArchiveType(inputArchive []byte) (*ArchiveType, error) {

	return nil, fmt.Errorf("unknown filetype")
}

type ArchiveType struct {
	Algorithm  string
	MagicBytes []byte
	Name       string
	Offset     int
	// Extract Pointer
}

var KnownTypes = []ArchiveType{
	{
		Algorithm:  "Lempel-Ziv-Welch",
		Name:       "zip",
		Offset:     0,
		MagicBytes: []byte{0x1F, 0x9D},
	},
}

func (at *ArchiveType) FileHeaderSize() int {
	return at.Offset + len(at.MagicBytes)
}

func MaxArchiveHeaderLength() int {
	bufferSize := 0
	for _, at := range KnownTypes {
		signatureLen := (at.Offset + len(at.MagicBytes))
		if signatureLen > bufferSize {
			bufferSize = signatureLen
		}
	}
	return bufferSize
}

func verifyPathPrefix(pathPrefix, path string) error {
	if !strings.HasPrefix(path, pathPrefix) {
		return fmt.Errorf("path prefix missmatch")
	}
	return nil
}
