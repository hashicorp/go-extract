package extract

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-extract/extractor"
)

func TestFindExtractor(t *testing.T) {
	// test cases
	cases := []struct {
		name     string
		input    string
		expected Extractor
	}{
		{
			name:     "get zip extractor from file",
			input:    "foo.zip",
			expected: extractor.NewZip(nil),
		},
		{
			name:     "get zip extractor from file in path",
			input:    "foo.zip",
			expected: extractor.NewZip(nil),
		},
		{
			name:     "get tar extractor from file",
			input:    "foo.tar",
			expected: extractor.NewTar(nil),
		},
		{
			name:     "get tar extractor from file in path",
			input:    "foo.tar",
			expected: extractor.NewTar(nil),
		},
		{
			name:     "unspported file type .7z",
			input:    "foo.7z",
			expected: nil,
		},
		{
			name:     "no filetype",
			input:    "foo",
			expected: nil,
		},
		{
			name:     "camel case",
			input:    "foo.zIp",
			expected: extractor.NewZip(nil),
		},
		{
			name:     "camel case",
			input:    "foo.TaR",
			expected: extractor.NewTar(nil),
		},
	}

	// run cases
	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			// prepare vars
			var failed bool
			want := tc.expected

			// perform actual tests
			got := findExtractor(tc.input)

			// success if both are nil and no engine found
			if want == got {
				return
			}

			// check if engine detection failed
			if got == nil {
				failed = true
			}

			// if not failed yet, compare identified suffixes
			if !failed {
				if got.FileSuffix() != want.FileSuffix() {
					failed = true
				}
			}

			if failed {
				t.Errorf("test case %d failed: %s\nexpected: %v\ngot: %v", i, tc.name, want, got)
			}

		})
	}

}
