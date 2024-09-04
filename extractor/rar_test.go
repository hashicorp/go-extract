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

var testRarArchiveBase64 = "UmFyIRoHAQAzkrXlCgEFBgAFAQGAgAADk1YoJQIDC50ABJ0ApIMClAgA9IAAAQdkaXIvZm9vCgMTQPjXZsjBSQhNaSAgNCBTZXAgMjAyNCAwODowMzo0NCBDRVNUCpQdu+oiAgMLnQAEnQCkgwI+z7uqgAABBGZpbGUKAxPEDddmxHsQDkRpICAzIFNlcCAyMDI0IDE1OjIzOjE2IENFU1QKe1xvKCwCAxcABAftwwIAAAAAgAABBGxpbmsKAxNM+NdmSCZHGAsFAQAHZGlyL2Zvb0A2hh0bAgMLAAEA7YMBgAABA2RpcgoDE0D412Z533kHHXdWUQMFBAA="

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
		log.Fatalf("error decoding base64 string: %v", err)
	}
	archiveReader := bytes.NewReader(archiveBytes)

	// Create a temporary directory and unpack the Rar archive
	ctx := context.Background()
	target := target.NewOS()
	tmpDir := t.TempDir()
	cfg := config.NewConfig()
	err = UnpackRar(ctx, target, tmpDir, archiveReader, cfg)
	if err != nil {
		t.Fatalf("error unpacking rar archive: %v", err)
	}

	// reset the reader
	_, err = archiveReader.Seek(0, 0)
	if err != nil {
		t.Fatalf("error resetting reader: %v", err)
	}

	// Create a temporary directory and unpack the Rar archive with cached in memory
	tmpDir = t.TempDir()
	cfgCachedInMemory := config.NewConfig(config.WithCacheInMemory(true))
	err = UnpackRar(ctx, target, tmpDir, archiveReader, cfgCachedInMemory)
	if err != nil {
		t.Fatalf("error unpacking rar archive: %v", err)
	}

}
