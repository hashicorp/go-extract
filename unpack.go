package extract

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
	"github.com/hashicorp/go-extract/target"
)

// Unpack reads data from src, identifies if its a known archive type. If so, dst is unpacked
// in dst. opts can be given to adjust the config and target.
func Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
	var ex Extractor

	header, err := extractor.NewHeaderReader(src, extractor.MaxHeaderLength)
	if err != nil {
		return err
	}
	headerData := header.PeekHeader()

	if ex = findExtractor(headerData); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// perform extraction with identified reader
	return ex.Unpack(ctx, header, dst, t, c)
}

// findExtractor identifies the correct extractor based on magic bytes.
func findExtractor(data []byte) Extractor {
	// find extractor with longest suffix match
	for _, ex := range extractor.ExtractorsForMagicBytes {
		// check all possible magic bytes for extract engine
		for _, magicBytes := range ex.MagicBytes {

			// check for byte match
			if matchesMagicBytes(data, ex.Offset, magicBytes) {
				return ex.NewExtractor()
			}
		}
	}

	// no matching reader found
	return nil
}

// matchesMagicBytes checks if the bytes in data are equal to magicBytes after at a given offset
func matchesMagicBytes(data []byte, offset int, magicBytes []byte) bool {
	if offset+len(magicBytes) > len(data) {
		return false
	}

	return bytes.Equal(magicBytes, data[offset:offset+len(magicBytes)])
}
