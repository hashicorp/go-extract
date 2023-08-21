package extract

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
)

func Unpack(ctx context.Context, src string, dst string, opts ...ExtractorOption) error {

	var ex Extractor
	if ex = findExtractor(src); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	for _, opt := range opts {
		opt(&ex)
	}

	// check if extraction timeout is set
	if ex.Config().MaxExtractionTime == -1 {
		if err := ex.Unpack(ctx, src, dst); err != nil {
			return err
		}
	} else {
		if err := extractWithTimeout(ctx, ex, src, dst); err != nil {
			return err
		}
	}

	return nil

}

func WithMaxFiles(maxFiles int64) ExtractorOption {
	return func(e *Extractor) {
		(*e).Config().MaxFiles = maxFiles
	}
}

func WithMaxFileSize(maxFileSize int64) ExtractorOption {
	return func(e *Extractor) {
		(*e).Config().MaxFileSize = maxFileSize
	}
}

func WithMaxExtractionTime(maxExtractionTime int64) ExtractorOption {
	return func(e *Extractor) {
		(*e).Config().MaxExtractionTime = maxExtractionTime
	}
}

func WithOverwrite() ExtractorOption {
	return func(e *Extractor) {
		(*e).Config().Overwrite = true
	}
}

// findExtractor identifies the correct extractor based on src filename with longest suffix match
func findExtractor(src string) Extractor {

	// TODO(jan): detect filetype based on magic bytes

	// generate config
	config := config.NewConfig()

	// Prepare available extractors
	extractors := []Extractor{extractor.NewTar(config), extractor.NewZip(config)}

	// find extractor with longest suffix match
	var maxSuffixLength int
	var engine Extractor
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
func extractWithTimeout(ctx context.Context, ex Extractor, src string, dst string) error {
	// prepare extraction process
	exChan := make(chan error, 1)
	go func() {
		// extract files in tmpDir
		if err := ex.Unpack(ctx, src, dst); err != nil {
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
	case <-time.After(time.Duration(ex.Config().MaxExtractionTime) * time.Second):
		return fmt.Errorf("maximum extraction time exceeded")
	}

	return nil
}
