package extractor

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

func TestMatchesMagicBytes(t *testing.T) {
	cases := []struct {
		name        string
		data        []byte
		magicBytes  [][]byte
		offset      int
		expectMatch bool
	}{
		{
			name:        "match",
			data:        []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			magicBytes:  [][]byte{{0x02, 0x03}},
			offset:      2,
			expectMatch: true,
		},
		{
			name:        "missmatch",
			data:        []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			magicBytes:  [][]byte{{0x02, 0x03}},
			offset:      1,
			expectMatch: false,
		},
		{
			name:        "to few data to match",
			data:        []byte{0x00},
			magicBytes:  [][]byte{{0x02, 0x03}},
			offset:      1,
			expectMatch: false,
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create testing directory
			expected := tc.expectMatch
			got := matchesMagicBytes(tc.data, tc.offset, tc.magicBytes)

			// success if both are nil and no engine found
			if got != expected {
				t.Errorf("test case %d failed: %s!", i, tc.name)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	c := config.NewConfig(config.WithContinueOnError(false))
	metrics := &config.Metrics{}

	err := errors.New("test error")
	handleError(c, metrics, "test message", err)

	if metrics.ExtractionErrors != int64(1) {
		t.Error("ExtractionErrors was not incremented")
	}

	if metrics.LastExtractionError.Error() != "test message: test error" {
		t.Error("LastExtractionError was not set correctly")
	}

	c = config.NewConfig(config.WithContinueOnError(true))
	err = handleError(c, metrics, "test message", err)
	if err != nil {
		t.Error("handleError should return nil when continueOnError is true")
	}
}
