package extractor

import (
	"archive/tar"
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
	"github.com/hashicorp/go-extract/telemetry"
)

// OffsetTar is the offset where the magic bytes are located in the file
const OffsetTar = 257

// FileExtensionTar is the file extension for tar files
const FileExtensionTar = "tar"

// MagicBytesTar are the magic bytes for tar files
var MagicBytesTar = [][]byte{
	[]byte("ustar\x00tar\x00"),
	[]byte("ustar\x00"),
	[]byte("ustar  \x00"),
}

// HeaderSizeTar is the size of the header for tar files
// I know it is ugly, but I am trying to keep the code as clean as possible
var headerSizeTar = OffsetTar + len(MagicBytesTar[0])

// IsTar checks if the header matches the magic bytes for tar files
func IsTar(data []byte) bool {
	return matchesMagicBytes(data, OffsetTar, MagicBytesTar)
}

// Unpack sets a timeout for the ctx and starts the tar extraction from src to dst.
func UnpackTar(ctx context.Context, t target.Target, dst string, src io.Reader, cfg *config.Config) error {

	// prepare telemetry capturing
	td := &telemetry.Data{ExtractedType: FileExtensionTar}
	defer cfg.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// prepare reader
	limitedReader := newLimitErrorReader(src, cfg.MaxInputSize())
	defer captureInputSize(td, limitedReader)

	// start extraction
	return unpackTar(ctx, t, limitedReader, dst, cfg, td)
}

// unpackTar extracts the tar archive from src to dst
func unpackTar(ctx context.Context, t target.Target, src io.Reader, dst string, c *config.Config, td *telemetry.Data) error {
	return extract(ctx, t, dst, &tarWalker{tr: tar.NewReader(src)}, c, td)
}

// tarWalker is a walker for tar files
type tarWalker struct {
	tr *tar.Reader
}

// Type returns the file extension for tar files
func (t *tarWalker) Type() string {
	return FileExtensionTar
}

// Next returns the next entry in the tar archive
func (t *tarWalker) Next() (archiveEntry, error) {
	hdr, err := t.tr.Next()
	if err != nil {
		return nil, err
	}
	return &tarEntry{hdr, t.tr}, nil
}

// tarEntry is an entry in a tar archive
type tarEntry struct {
	hdr *tar.Header
	tr  *tar.Reader
}

// Name returns the name of the entry
func (t *tarEntry) Name() string {
	return t.hdr.Name
}

// Size returns the size of the entry
func (t *tarEntry) Size() int64 {
	return t.hdr.Size
}

// Mode returns the mode of the entry
func (t *tarEntry) Mode() os.FileMode {
	return t.hdr.FileInfo().Mode()
}

// Linkname returns the linkname of the entry
func (t *tarEntry) Linkname() string {
	return t.hdr.Linkname
}

// IsRegular returns true if the entry is a regular file
func (t *tarEntry) IsRegular() bool {
	return t.hdr.Typeflag == tar.TypeReg
}

// IsDir returns true if the entry is a directory
func (t *tarEntry) IsDir() bool {
	return t.hdr.Typeflag == tar.TypeDir
}

// IsSymlink returns true if the entry is a symlink
func (t *tarEntry) IsSymlink() bool {
	return t.hdr.Typeflag == tar.TypeSymlink
}

// Open returns a reader for the entry
func (t *tarEntry) Open() (io.ReadCloser, error) {
	return &noopReaderCloser{t.tr}, nil
}

// Type returns the type of the entry
func (t *tarEntry) Type() fs.FileMode {
	return fs.FileMode(t.hdr.Typeflag)
}
