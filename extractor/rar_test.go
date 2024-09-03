package extractor

import (
	"bytes"
	"context"
	"encoding/base64"
	"log"
	"testing"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

var testRarArchiveBase64 = "UmFyIRoHAQAzkrXlCgEFBgAFAQGAgACUHbvqIgIDC50ABJ0ApIMCPs+7qoAAAQRmaWxlCgMTxA3XZsR7EA5EaSAgMyBTZXAgMjAyNCAxNToyMzoxNiBDRVNUCpbhsN0pAgMUAAQE7cMCAAAAAIAAAQRsaW5rCgMTyQ3XZizK2TQIBQEABGZpbGVVBY+/GwIDCwABAO2DAYAAAQNkaXIKAxO3DddmazZtHx13VlEDBQQA"

// TestIsRar tests the IsRar function
func TestIsRar(t *testing.T) {

	// test cases
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00}, false},
	}

	// run tests
	for _, tt := range tests {
		if got := IsRar(tt.header); got != tt.want {
			t.Errorf("IsRar(%v) = %v; want %v", tt.header, got, tt.want)
		}
	}

}

// TestRarUnpacker tests the RarUnpacker function
func TestUnpackRar(t *testing.T) {

	// Decode the base64 string
	archiveBytes, err := base64.StdEncoding.DecodeString(testRarArchiveBase64)
	if err != nil {
		log.Fatalf("Error decoding base64 string: %v", err)
	}
	archiveReader := bytes.NewReader(archiveBytes)

	// Create a temporary directory and unpack the Rar archive
	ctx := context.Background()
	target := target.NewOS()
	cfg := config.NewConfig()
	tmpDir := t.TempDir()
	err = UnpackRar(ctx, target, tmpDir, archiveReader, cfg)
	if err != nil {
		t.Fatalf("Error unpacking Rar archive: %v", err)
	}

	// reset the reader
	archiveReader = bytes.NewReader(archiveBytes)

	// Create a temporary directory and unpack the Rar archive with cached in memory
	tmpDir = t.TempDir()
	cfgCachedInMemory := config.NewConfig(config.WithCacheInMemory(true))
	err = UnpackRar(ctx, target, tmpDir, archiveReader, cfgCachedInMemory)
	if err != nil {
		t.Fatalf("Error unpacking Rar archive: %v", err)
	}

}
