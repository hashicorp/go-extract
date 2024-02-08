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
// in dst. opts can be given to adjust the config.
func Unpack(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// default target
	target := target.NewOs()

	// perform extraction with identified reader
	return UnpackOnTarget(ctx, src, dst, target, c)
}

// UnpackOnTarget reads data from src on a givin target, identifies if its a known archive type. If so, dst is unpacked
// in dst. opts can be given to adjust the config.
func UnpackOnTarget(ctx context.Context, src io.Reader, dst string, trgt target.Target, c *config.Config) error {

	// read headerReader to identify archive type
	header, reader, err := getHeader(src)
	if err != nil {
		return err
	}

	// find extractor for header
	var ex Extractor
	if ex = findExtractor(header); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	// perform extraction with identified reader
	return ex.Unpack(ctx, reader, dst, trgt, c)
}

// getHeader reads the header from src and returns it. If src is a io.Seeker, the header is read
// directly from the reader and the reader gets reset. If src is not a io.Seeker, the header is read
// and transformed into a HeaderReader, which is returned as the second return value. If an error
// occurs, the header is nil and the error is returned as the third return value
func getHeader(src io.Reader) ([]byte, io.Reader, error) {

	// allocate buffer for header
	header := make([]byte, extractor.MaxHeaderLength)

	// check if source offers seek and preserve type of source
	if s, ok := src.(io.Seeker); ok {
		// read header from source
		_, err := src.Read(header)
		if err != nil {
			return nil, nil, err
		}
		// reset reader
		_, err = s.Seek(0, io.SeekStart)
		if err != nil {
			return nil, nil, err
		}
		return header, src, nil
	}

	headerReader, err := extractor.NewHeaderReader(src, extractor.MaxHeaderLength)
	if err != nil {
		return nil, nil, err
	}
	return headerReader.PeekHeader(), headerReader, nil
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
