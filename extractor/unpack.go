package extractor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-extract"
)

func Unpack(ctx context.Context, src string, dst string) error {
	config := extract.Default()
	return UnpackWithConfig(ctx, config, src, dst)
}

// Unpack extracts archive supplied in src to dst.
func UnpackWithConfig(ctx context.Context, config *extract.Config, src string, dst string) error {

	// identify extraction engine
	var ex extract.Extractor
	if ex = findExtractor(config, src); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// create tmp directory
	// TODO(jan): check if tmpDir needed
	tmpDir, err := os.MkdirTemp(os.TempDir(), "extract*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	tmpDir = filepath.Clean(tmpDir) + string(os.PathSeparator)

	// check if extraction timeout is set
	if config.MaxExtractionTime == -1 {
		if err := ex.Unpack(src, dst); err != nil {
			return err
		}
	} else {
		if err := extractWithTimeout(config, ex, src, tmpDir); err != nil {
			return err
		}
	}

	// move content from tmpDir to destination
	if err := CopyDirectory(tmpDir, dst); err != nil {
		return err
	}

	return nil
}

// findExtractor identifies the correct extractor based on src filename with longest suffix match
func findExtractor(config *extract.Config, src string) extract.Extractor {

	// TODO(jan): detect filetype based on magic bytes

	// Prepare available extractors
	extractors := []extract.Extractor{NewTar(config), NewZip(config)}

	// find extractor with longest suffix match
	var maxSuffixLength int
	var engine extract.Extractor
	for _, ex := range extractors {

		// get suffix
		suff := ex.FileSuffix()

		// skip non-matching extractors
		if !strings.HasSuffix(strings.ToLower(src), suff) {
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

// extractWithTimeout extracts src with supplied extractor ex to dst
func extractWithTimeout(config *extract.Config, ex extract.Extractor, src string, dst string) error {
	// prepare extraction process
	exChan := make(chan error, 1)
	go func() {
		// extract files in tmpDir
		if err := ex.Unpack(src, dst); err != nil {
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
	case <-time.After(time.Duration(config.MaxExtractionTime) * time.Second):
		return fmt.Errorf("maximum extraction time exceeded")
	}

	return nil
}
