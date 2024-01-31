package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// extractor is a private interface and defines all functions that needs to be implemented by an extraction engine.
type extractor interface {
	// Unpack is the main entrypoint to an extraction engine that takes the contents from src and extracts them to dst.
	Unpack(ctx context.Context, src io.Reader, dst string, target target.Target, config *config.Config) error
}

// prepare ensures limited read and generic metric capturing
// remark: this preparation is located in the extractor package so that the
// different extractor engines can be used independently and keep their
// functionality.
func prepare(ctx context.Context, src io.Reader, c *config.Config) io.Reader {

	// setup reader and timer
	start := time.Now()                                      // capture start to calculate execution time
	ler := newLimitErrorReaderCounter(src, c.MaxInputSize()) // ensure input size and capture metrics

	// extend metric collection
	c.AddMetricsProcessor(func(ctx context.Context, m *config.Metrics) {
		m.ExtractionDuration = time.Since(start) // capture execution time
		m.InputSize = int64(ler.ReadBytes())     // capture inputSize metric
	})

	return ler
}

// AvailableExtractors is collection of new extractor functions with
// the required magic bytes and potential offset
var AvailableExtractors = []struct {
	NewExtractor func() extractor
	HeaderCheck  func([]byte) bool
	MagicBytes   [][]byte
	Offset       int
}{
	{
		NewExtractor: func() extractor {
			return NewTar()
		},
		HeaderCheck: IsTar,
		MagicBytes:  magicBytesTar,
		Offset:      offsetTar,
	},
	{
		NewExtractor: func() extractor {
			return NewZip()
		},
		HeaderCheck: IsZip,
		MagicBytes:  magicBytesZIP,
	},
	{
		NewExtractor: func() extractor {
			return NewGzip()
		},
		HeaderCheck: IsGZIP,
		MagicBytes:  magicBytesGZIP,
	},
}

var MaxHeaderLength int

// init calculates the maximum header length
func init() {
	for _, ex := range AvailableExtractors {
		needs := ex.Offset
		for _, mb := range ex.MagicBytes {
			if len(mb)+ex.Offset > needs {
				needs = len(mb) + ex.Offset
			}
		}
		if needs > MaxHeaderLength {
			MaxHeaderLength = needs
		}
	}
}

func matchesMagicBytes(data []byte, offset int, magicBytes [][]byte) bool {
	// check all possible magic bytes until match is found
	for _, mb := range magicBytes {
		// check if header is long enough
		if offset+len(mb) > len(data) {
			continue
		}

		// check for byte match
		if bytes.Equal(mb, data[offset:offset+len(mb)]) {
			return true
		}
	}

	// no match found
	return false
}

// handleError increases the error counter, sets the latest error and
// decides if extraction should continue.
func handleError(c *config.Config, metrics *config.Metrics, msg string, err error) error {

	// increase error counter and set error
	metrics.ExtractionErrors++
	metrics.LastExtractionError = fmt.Errorf("%s: %s", msg, err)

	// do not end on error
	if c.ContinueOnError() {
		c.Logger().Error(msg, "error", err)
		return nil
	}

	// end extraction on error
	return metrics.LastExtractionError
}