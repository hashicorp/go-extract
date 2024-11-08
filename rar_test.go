package extract_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"log"
	"testing"

	"github.com/hashicorp/go-extract"
)

var testRarArchiveBase64 = "UmFyIRoHAQAzkrXlCgEFBgAFAQGAgAADk1YoJQIDC50ABJ0ApIMClAgA9IAAAQdkaXIvZm9vCgMTQPjXZsjBSQhNaSAgNCBTZXAgMjAyNCAwODowMzo0NCBDRVNUCpQdu+oiAgMLnQAEnQCkgwI+z7uqgAABBGZpbGUKAxPEDddmxHsQDkRpICAzIFNlcCAyMDI0IDE1OjIzOjE2IENFU1QKe1xvKCwCAxcABAftwwIAAAAAgAABBGxpbmsKAxNM+NdmSCZHGAsFAQAHZGlyL2Zvb0A2hh0bAgMLAAEA7YMBgAABA2RpcgoDE0D412Z533kHHXdWUQMFBAA="

func TestIsRar(t *testing.T) {
	tests := []struct {
		header []byte
		want   bool
	}{
		{[]byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00}, false},
	}

	for _, test := range tests {
		if got := extract.IsRar(test.header); got != test.want {
			t.Errorf("IsRar(%v) = %v; want %v", test.header, got, test.want)
		}
	}

}

func TestUnpackRar(t *testing.T) {
	// Decode the base64 string
	archiveBytes, err := base64.StdEncoding.DecodeString(testRarArchiveBase64)
	if err != nil {
		log.Fatalf("error decoding base64 string: %v", err)
	}
	archiveReader := bytes.NewReader(archiveBytes)

	// Create a temporary directory and unpack the Rar archive
	ctx := context.Background()
	target := extract.NewDisk()
	tmpDir := t.TempDir()
	cfg := extract.NewConfig(extract.WithContinueOnUnsupportedFiles(true))
	err = extract.UnpackRar(ctx, target, tmpDir, archiveReader, cfg)
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
	cfgCachedInMemoryIgnoreSymlink := extract.NewConfig(
		extract.WithCacheInMemory(true),
		extract.WithContinueOnUnsupportedFiles(true))
	err = extract.UnpackRar(ctx, target, tmpDir, archiveReader, cfgCachedInMemoryIgnoreSymlink)
	if err != nil {
		t.Fatalf("error unpacking rar archive: %v", err)
	}

	// Create a temporary directory and unpack the Rar archive with cached in memory,
	// but fail due to the symlink in the archive
	tmpDir = t.TempDir()
	cfgCachedInMemoryFailOnSymlink := extract.NewConfig(
		extract.WithCacheInMemory(true),
		extract.WithContinueOnUnsupportedFiles(false))
	err = extract.UnpackRar(ctx, target, tmpDir, archiveReader, cfgCachedInMemoryFailOnSymlink)
	if err == nil {
		t.Fatalf("expected error unpacking symlink from rar archive, but got nil")
	}
}
