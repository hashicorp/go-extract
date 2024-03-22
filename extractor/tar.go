package extractor

import (
	"archive/tar"
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

// offsetTar is the offset where the magic bytes are located in the file
const offsetTar = 257

// fileExtensionTar is the file extension for tar files
var fileExtensionTar = "tar"

// magicBytesTar are the magic bytes for tar files
var magicBytesTar = [][]byte{
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
	{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x20, 0x00},
}

// IsTar checks if the header matches the magic bytes for tar files
func IsTar(data []byte) bool {
	return matchesMagicBytes(data, offsetTar, magicBytesTar)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func UnpackTar(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// prepare telemetry capturing
	td := &telemetry.Data{ExtractedType: fileExtensionTar}
	defer c.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// prepare reader
	limitedReader := NewLimitErrorReader(src, c.MaxInputSize())
	defer captureInputSize(td, limitedReader)

	// start extraction
	return unpackTar(ctx, limitedReader, dst, c, td)
}

func unpackTar(ctx context.Context, src io.Reader, dst string, c *config.Config, td *telemetry.Data) error {
	return extract(ctx, &tarWalker{tr: tar.NewReader(src)}, dst, c, td)
}

type tarWalker struct {
	tr *tar.Reader
}

func (t *tarWalker) Type() string {
	return fileExtensionTar
}

func (t *tarWalker) Next() (archiveEntry, error) {
	hdr, err := t.tr.Next()
	if err != nil {
		return nil, err
	}
	return &tarEntry{hdr, t.tr}, nil
}

type tarEntry struct {
	hdr *tar.Header
	tr  *tar.Reader
}

func (t *tarEntry) Name() string {
	return t.hdr.Name
}

func (t *tarEntry) Size() int64 {
	return t.hdr.Size
}

func (t *tarEntry) Mode() os.FileMode {
	return t.hdr.FileInfo().Mode()
}

func (t *tarEntry) Linkname() string {
	return t.hdr.Linkname
}

func (t *tarEntry) IsRegular() bool {
	return t.hdr.Typeflag == tar.TypeReg
}

func (t *tarEntry) IsDir() bool {
	return t.hdr.Typeflag == tar.TypeDir
}

func (t *tarEntry) IsSymlink() bool {
	return t.hdr.Typeflag == tar.TypeSymlink
}

func (t *tarEntry) Open() (io.ReadCloser, error) {
	return &NoopReaderCloser{t.tr}, nil
}

func (t *tarEntry) Type() fs.FileMode {
	return fs.FileMode(t.hdr.Typeflag)
}
