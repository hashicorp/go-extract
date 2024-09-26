package extractor

import (
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
	"github.com/hashicorp/go-extract/telemetry"
	"github.com/nwaples/rardecode"
)

// FileExtensionRar is the file extension for Rar files
const FileExtensionRar = "rar"

// magicBytesRar are the magic bytes for Rar files
var magicBytesRar = [][]byte{
	{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00},       // Rar 1.5
	{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00}, // Rar 5.0
}

// IsRar checks if the header matches the magic bytes for Rar files
func IsRar(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytesRar)
}

// UnpackRar sets a timeout for the ctx and starts the Rar extraction from src to dst.
func UnpackRar(ctx context.Context, t target.Target, dst string, src io.Reader, cfg *config.Config) error {

	// prepare telemetry data collection and emit
	td := &telemetry.Data{ExtractedType: FileExtensionRar}
	defer cfg.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// ensure that all bytes are read from the reader
	cachedReader, cached, err := ReaderToCache(cfg, src)
	if err != nil {
		return handleError(cfg, td, "cannot cache reader", err)
	}
	defer func() {
		if !cached {
			return
		}
		if f, ok := cachedReader.(*os.File); ok {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
	}()

	return unpackRar(ctx, t, dst, cachedReader, cfg, td)
}

// unpackRar extracts a Rar archive from src to dst.
func unpackRar(ctx context.Context, t target.Target, dst string, src io.Reader, cfg *config.Config, td *telemetry.Data) error {

	// log extraction
	cfg.Logger().Info("extracting rar")

	// check if src is a file, instantiate reader from file
	if s, ok := src.(*os.File); ok {
		a, err := rardecode.OpenReader(s.Name(), "")
		if err != nil {
			return handleError(cfg, td, "cannot create rar decoder", err)
		}
		defer a.Close()
		return extract(ctx, t, dst, &rarWalker{&a.Reader}, cfg, td)
	}

	// get bytes from reader
	a, err := rardecode.NewReader(src, "")
	if err != nil {
		return handleError(cfg, td, "cannot create rar decoder", err)
	}
	return extract(ctx, t, dst, &rarWalker{a}, cfg, td)
}

// rarWalker is an archiveWalker for Rar files
type rarWalker struct {
	r *rardecode.Reader
}

// Type returns the file extension for rar files
func (rw *rarWalker) Type() string {
	return FileExtensionRar
}

// Next returns the next entry in the rar file
func (rw *rarWalker) Next() (archiveEntry, error) {
	fh, err := rw.r.Next()
	if err != nil {
		return nil, err
	}
	re := &rarEntry{fh, rw.r}
	if re.IsSymlink() { // symlink not supported
		return nil, UnsupportedFile(re.Name())
	}
	return re, nil
}

// rarEntry is an archiveEntry for Rar files
type rarEntry struct {
	f *rardecode.FileHeader
	r io.Reader
}

// Name returns the name of the file
func (re *rarEntry) Name() string {
	return re.f.Name
}

// Size returns the size of the file
func (re *rarEntry) Size() int64 {
	return re.f.UnPackedSize
}

// Mode returns the mode of the file
func (z *rarEntry) Mode() os.FileMode {
	return z.f.Mode()
}

// Linkname symlinks are not supported
func (re *rarEntry) Linkname() string {
	return ""
}

// IsRegular returns true if the file is a regular file
func (re *rarEntry) IsRegular() bool {
	return re.f.Mode().IsRegular()
}

// IsDir returns true if the file is a directory
func (z *rarEntry) IsDir() bool {
	return z.f.IsDir
}

// IsSymlink returns true if the file is a symlink
func (z *rarEntry) IsSymlink() bool {
	return z.f.Mode()&fs.ModeSymlink != 0
}

// Type returns the type of the file
func (z *rarEntry) Type() fs.FileMode {
	return z.f.Mode().Type()
}

// Open returns a reader for the file
func (z *rarEntry) Open() (io.ReadCloser, error) {
	return io.NopCloser(z.r), nil
}
