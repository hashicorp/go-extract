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
	target := target.NewOS()

	// perform extraction with identified reader
	return UnpackOnTarget(ctx, src, dst, target, c)
}

// UnpackOnTarget reads data from src on a givin target, identifies if its a known archive type. If so, dst is unpacked
// in dst. opts can be given to adjust the config.
func UnpackOnTarget(ctx context.Context, src io.Reader, dst string, tgt target.Target, c *config.Config) error {

	// read headerReader to identify archive type
	header, reader, err := getHeader(src)
	if err != nil {
		return fmt.Errorf("failed to read header: %s", err)
	}

	// find extractor for header
	var ex Extractor
	if ex = findExtractor(header); ex == nil {
		return fmt.Errorf("archive type not supported")
	}

	switch ex.(type) {
	case *extractor.Tar:
		c.Logger().Info("extracting tar")
	case *extractor.Zip:
		c.Logger().Info("extracting zip")
	case *extractor.Gzip:
		c.Logger().Info("extracting gzip")
	}

	// perform extraction with identified reader
	return ex.Unpack(ctx, reader, dst, tgt, c)
}

// getHeader reads the header from src and returns it. If src is a io.Seeker, the header is read
// directly from the reader and the reader gets reset. If src is not a io.Seeker, the header is read
// and transformed into a HeaderReader, which is returned as the second return value. If an error
// occurs, the header is nil and the error is returned as the third return value
func getHeader(src io.Reader) ([]byte, io.Reader, error) {

	// check if source offers seek and preserve type of source
	if s, ok := src.(io.Seeker); ok {

		// allocate buffer for header
		header := make([]byte, extractor.MaxHeaderLength)

		// read header from source
		_, err := src.Read(header)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read header: %s", err)
		}
		// reset reader
		_, err = s.Seek(0, io.SeekStart)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to reset reader: %s", err)
		}
		return header, src, nil
	}

	headerReader, err := extractor.NewHeaderReader(src, extractor.MaxHeaderLength)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create header reader: %s", err)
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
