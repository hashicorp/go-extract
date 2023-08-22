package extract

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
)

func Unpack(ctx context.Context, src string, dst string, opts ...ExtractorOption) error {
	var ex Extractor

	if ex = findExtractor(src); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// apply extract options
	for _, opt := range opts {
		opt(&ex)
	}

	return ex.Unpack(ctx, src, dst)
}

// findExtractor identifies the correct extractor based on src filename with longest suffix match
func findExtractor(src string) Extractor {

	// prepare config
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
