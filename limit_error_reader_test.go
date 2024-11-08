package extract

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
			expectN:    4,
			wantErr:    false,
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			l := NewLimitErrorReader(r, test.limit)
			buf := make([]byte, test.bufferSize)
			n, err := l.Read(buf)
			if (err != nil) != test.wantErr {
				t.Fatalf("Read() error = %v, wantErr %v", err, test.wantErr)
			}
			if n != test.expectN {
				t.Errorf("Read() = %v, want %v", n, test.expectN)
			}
			if l.ReadBytes() != test.expectN {
				t.Errorf("ReadBytes() = %v, want %v", l.ReadBytes(), test.expectN)
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
			expectN: 4,
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			l := NewLimitErrorReader(r, test.limit)

			buf := make([]byte, len(test.input))
			n, err := l.Read(buf)
			if (err != nil) != test.wantErr {
				t.Fatalf("Read() error = %v, wantErr %v", err, test.wantErr)
			}
			if n != test.expectN {
				t.Errorf("Read() = %v, want %v", n, test.expectN)
			}
			if l.ReadBytes() != test.expectN {
				t.Errorf("ReadBytes() = %v, want %v", l.ReadBytes(), test.expectN)
			}
		})
	}
}
