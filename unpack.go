package extract

import (
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

	// read headerReader to identify archive type
	headerReader, err := extractor.NewHeaderReader(src, extractor.MaxHeaderLength)
	if err != nil {
		return err
	}
	var ex Extractor
	headerData := headerReader.PeekHeader()
	if ex = findExtractor(headerData); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// perform extraction with identified reader
	return ex.Unpack(ctx, headerReader, dst, t, c)
}

// findExtractor identifies the correct extractor based on magic bytes.
func findExtractor(data []byte) Extractor {
	// find extractor with longest suffix match
	for _, ex := range extractor.AvailableExtractors {
		if ex.HeaderCheck(data) {
			return ex.NewExtractor()
		}
	}

	// no matching reader found
	return nil
}
