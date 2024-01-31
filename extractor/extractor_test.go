package extractor

import (
	"strings"
	"testing"
)

// TestLimitErrorReader_Read tests the implementation of limitErrorReader.Read
func TestLimitErrorReader_Read(t *testing.T) {
	tests := []struct {
		name    string
		limit   int64
		input   string
		expectN int
		wantErr bool
	}{
		{
			name:    "Under limit",
			limit:   10,
			input:   "12345",
			expectN: 5,
			wantErr: false,
		},
		{
			name:    "At limit",
			limit:   5,
			input:   "12345",
			expectN: 5,
			wantErr: false,
		},
		{
			name:    "Over limit",
			limit:   4,
			input:   "12345",
			expectN: 5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			l := newLimitErrorReaderCounter(r, tt.limit)

			buf := make([]byte, len(tt.input))
			n, err := l.Read(buf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
			if n != tt.expectN {
				t.Errorf("Read() = %v, want %v", n, tt.expectN)
			}
			if l.ReadBytes() != tt.expectN {
				t.Errorf("ReadBytes() = %v, want %v", l.ReadBytes(), tt.expectN)
			}
		})
	}
}

func TestReadBytes(t *testing.T) {

	tests := []struct {
		name       string
		limit      int64
		input      string
		bufferSize int
		expectN    int
		wantErr    bool
	}{
		{
			name:       "Under limit",
			limit:      10,
			input:      "12345",
			bufferSize: 5,
			expectN:    5,
			wantErr:    false,
		},
		{
			name:       "At limit",
			limit:      5,
			input:      "12345",
			bufferSize: 5,
			expectN:    5,
			wantErr:    false,
		},
		{
			name:       "Over limit",
			limit:      4,
			input:      "12345",
			bufferSize: 5,
			expectN:    5,
			wantErr:    true,
		},
		{
			name:       "Under limit with buffer",
			limit:      10,
			input:      "12345",
			bufferSize: 2,
			expectN:    2,
			wantErr:    false,
		},
		{
			name:       "Unlimited",
			limit:      -1,
			input:      "12345",
			bufferSize: 5,
			expectN:    5,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			l := newLimitErrorReaderCounter(r, tt.limit)
			buf := make([]byte, tt.bufferSize)
			n, err := l.Read(buf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
			if n != tt.expectN {
				t.Errorf("Read() = %v, want %v", n, tt.expectN)
			}
			if l.ReadBytes() != tt.expectN {
				t.Errorf("ReadBytes() = %v, want %v", l.ReadBytes(), tt.expectN)
			}
		})
	}
}

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
