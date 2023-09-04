package extract

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
)

// Unpack reads data from src, identifies if its a known archive type. If so, dst is unpackecked
// in dst. opts can be given to adjust the config and target.
func Unpack(ctx context.Context, src io.Reader, dst string, opts ...ExtractorOption) error {
	var ex Extractor

	// get bytes
	archiveData, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	if ex = findExtractor(archiveData); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// apply extract options
	for _, opt := range opts {
		opt(&ex)
	}

	// perform extraction with identified reader
	return ex.Unpack(ctx, bytes.NewReader(archiveData), dst)
}

// findExtractor identifies the correct extractor based on src filename with longest suffix match
func findExtractor(data []byte) Extractor {

	// prepare config
	config := config.NewConfig()

	// Prepare available extractors
	extractors := []Extractor{extractor.NewTar(config), extractor.NewZip(config), extractor.NewGzip(config)}

	// find extractor with longest suffix match
	for _, ex := range extractors {

		// get suffix
		offset := ex.Offset()

		// check all possible magic bytes for extract engine
		for _, magicBytes := range ex.MagicBytes() {

			// check for byte match
			if matchesMagicBytes(data, offset, magicBytes) {
				return ex
			}
		}
	}

	// no matching reader found
	return nil
}

// matchesMagicBytes checks if the bytes in data are equal to magicBytes after at a given offset
func matchesMagicBytes(data []byte, offset int, magicBytes []byte) bool {

	// first, check the length
	if offset+len(magicBytes) > len(data) {
		return false
	}

	// compare magic bytes with bytes in data
	for idx, fileByte := range data[offset : offset+len(magicBytes)] {
		if fileByte != magicBytes[idx] {
			return false
		}
	}

	// similar if no missmatch found
	return true
}
