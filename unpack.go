package extract

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
)

func Unpack(ctx context.Context, src io.Reader, dst string, opts ...ExtractorOption) error {
	var ex Extractor

	// get bytes
	archive, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	if ex = findExtractor(archive); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// apply extract options
	for _, opt := range opts {
		opt(&ex)
	}

	return ex.Unpack(ctx, bytes.NewReader(archive), dst)
}

// findExtractor identifies the correct extractor based on src filename with longest suffix match
func findExtractor(data []byte) Extractor {

	// prepare config
	config := config.NewConfig()

	// Prepare available extractors
	extractors := []Extractor{extractor.NewTar(config), extractor.NewZip(config)}

	// find extractor with longest suffix match
	for _, ex := range extractors {

		// get suffix
		offset := ex.Offset()

		for _, magicBytes := range ex.MagicBytes() {

			// compare magic bytes with readed bytes
			var missMatch bool
			for idx, fileByte := range data[offset : offset+len(magicBytes)] {
				if fileByte != magicBytes[idx] {
					missMatch = true
					break
				}
			}

			// if no missmatch, successfull identified engine!
			if !missMatch {
				return ex
			}

		}
	}

	// no matching reader found
	return nil
}
