package extractor

import (
	"strings"
	"testing"
)

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
