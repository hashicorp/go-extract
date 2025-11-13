// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build unix

package extract

import (
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestUnixTimeval(t *testing.T) {
	tests := []struct {
		input time.Time
		want  unix.Timeval
	}{
		{
			time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			unix.Timeval{Sec: 0, Usec: 0},
		},
		{
			// Note: the single nanosecond is rounded up to the next microsecond.
			time.Date(1970, 1, 1, 0, 0, 0, 1, time.UTC),
			unix.Timeval{Sec: 0, Usec: 1},
		},
		{
			// Note: the 100 nanoseconds are rounded up to the next microsecond.
			time.Date(1970, 1, 1, 0, 0, 0, 100, time.UTC),
			unix.Timeval{Sec: 0, Usec: 1},
		},
		{
			// Note: exactly 1 microsecond is not rounded up.
			time.Date(1970, 1, 1, 0, 0, 0, 1000, time.UTC),
			unix.Timeval{Sec: 0, Usec: 1},
		},
		{
			// Note: exactly 1 nanosecond past the microsecond is rounded up.
			time.Date(1970, 1, 1, 0, 0, 0, 1001, time.UTC),
			unix.Timeval{Sec: 0, Usec: 2},
		},
		{
			time.Date(1970, 1, 1, 0, 0, 1, 1000, time.UTC),
			unix.Timeval{Sec: 1, Usec: 1},
		},
		{
			time.Date(1970, 1, 1, 0, 0, 1, 2000, time.UTC),
			unix.Timeval{Sec: 1, Usec: 2},
		},
	}

	for _, test := range tests {
		t.Run(test.input.String(), func(t *testing.T) {
			got := unixTimeval(test.input)
			if got != test.want {
				t.Errorf("unixTimeval(%v) = %v; want %v", test.input, got, test.want)
			}
		})
	}
}
