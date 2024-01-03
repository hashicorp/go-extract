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

// extractorsForMagicBytes is collection of new extractor functions with
// the required magic bytes and potential offset
var extractorsForMagicBytes = []struct {
	newExtractor func() Extractor
	offset       int
	magicBytes   [][]byte
}{
	{
		newExtractor: func() Extractor {
			return extractor.NewTar()
		},
		offset:     extractor.OffsetTar,
		magicBytes: extractor.MagicBytesTar,
	},
	{
		newExtractor: func() Extractor {
			return extractor.NewZip()
		},
		magicBytes: extractor.MagicBytesZIP,
	},
	{
		newExtractor: func() Extractor {
			return extractor.NewGzip()
		},
		magicBytes: extractor.MagicBytesGZIP,
	},
}

// Unpack reads data from src, identifies if its a known archive type. If so, dst is unpacked
// in dst. opts can be given to adjust the config and target.
func Unpack(ctx context.Context, src io.Reader, dst string, t target.Target, c *config.Config) error {
	var ex Extractor

	// get bytes
	archiveData, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	if ex = findExtractor(archiveData); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// perform extraction with identified reader
	return ex.Unpack(ctx, bytes.NewReader(archiveData), dst, t, c)
}

// findExtractor identifies the correct extractor based on magic bytes.
func findExtractor(data []byte) Extractor {
	// find extractor with longest suffix match
	for _, ex := range extractorsForMagicBytes {
		// check all possible magic bytes for extract engine
		for _, magicBytes := range ex.magicBytes {

			// check for byte match
			if MatchesMagicBytes(data, ex.offset, magicBytes) {
				return ex.newExtractor()
			}
		}
	}

	// no matching reader found
	return nil
}

// MatchesMagicBytes checks if the bytes in data are equal to magicBytes after at a given offset
func MatchesMagicBytes(data []byte, offset int, magicBytes []byte) bool {
	if offset+len(magicBytes) > len(data) {
		return false
	}

	return bytes.Equal(magicBytes, data[offset:offset+len(magicBytes)])
}
