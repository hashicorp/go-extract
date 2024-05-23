package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-extract/config"
)

func TestSecurityCheckPath(t *testing.T) {

	cases := []struct {
		name        string
		basePath    string
		newDir      string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "legit directory name",
			basePath:    ".",
			newDir:      "test",
			config:      config.NewConfig(),
			expectError: false,
		},
		{
			name:        "traversal",
			basePath:    ".",
			newDir:      "../test",
			config:      config.NewConfig(),
			expectError: true,
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// perform actual test
			want := tc.expectError
			err := securityCheckPath(tc.config, tc.basePath, tc.newDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s", i, tc.name)
			}
		})
	}

	// test with symlinks
	// create testing directory with symlink to current dir
	testDir := t.TempDir()
	symlink := filepath.Join(testDir, "symlink")
	if err := os.Symlink(".", symlink); err != nil {
		t.Errorf(err.Error())
	}

	// perform actual test
	cases = []struct {
		name        string
		basePath    string
		newDir      string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "deny follow symlink",
			newDir:      filepath.Join("symlink", "deny"),
			config:      config.NewConfig(),
			expectError: true,
		},
		{
			name:        "allow follow symlink",
			newDir:      filepath.Join("symlink", "allow"),
			config:      config.NewConfig(config.WithFollowSymlinks(true)),
			expectError: false,
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.expectError
			err := securityCheckPath(tc.config, testDir, tc.newDir)
			got := err != nil
			if got != want {
				t.Errorf("test case %d failed: %s (%v)", i, tc.name, err)
			}
		})
	}

}
