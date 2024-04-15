package extractor

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

// magicBytesZIP contains the magic bytes for a zip archive.
// reference: https://golang.org/pkg/archive/zip/
var magicBytesZIP = [][]byte{
	{0x50, 0x4B, 0x03, 0x04},
}

// fileExtensionZIP is the file extension for zip files.
const fileExtensionZIP = "zip"

// IsZip checks if data is a zip archive. It returns true if data is a zip archive and false if data is not a zip archive.
func IsZip(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytesZIP)
}

// Unpack sets a timeout for the ctx and starts the zip extraction from src to dst. It returns an error if the extraction failed.
func UnpackZip(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// prepare telemetry data collection and emit
	td := &telemetry.Data{ExtractedType: fileExtensionZIP}
	defer c.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// check if src is a readerAt and an io.Seeker
	if sra, ok := src.(SeekerReaderAt); ok {
		return unpackZip(ctx, sra, dst, c, td)
	}

	// convert
	sra, err := ReaderToReaderAtSeeker(c, src)
	if err != nil {
		return handleError(c, td, "cannot convert reader to readerAt and seeker", err)
	}
	defer func() {
		if f, ok := src.(*os.File); ok {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	return unpackZip(ctx, sra, dst, c, td)
}

// unpackZip checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
// src is a readerAt and a seeker. If the InputSize exceeds the maximum input size, the function returns an error.
func unpackZip(ctx context.Context, src SeekerReaderAt, dst string, c *config.Config, m *telemetry.Data) error {

	// log extraction
	c.Logger().Info("extracting zip")

	// check if src is a seeker and readerAt
	s, _ := src.(io.Seeker)
	ra, _ := src.(io.ReaderAt)

	// get size of input and check if it exceeds maximum input size
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return handleError(c, m, "cannot seek to end of reader", err)
	}
	m.InputSize = size
	if c.MaxInputSize() != -1 && size > c.MaxInputSize() {
		return handleError(c, m, "cannot unpack zip", fmt.Errorf("input size exceeds maximum input size"))
	}

	// create zip reader and extract
	reader, err := zip.NewReader(ra, size)
	if err != nil {
		return handleError(c, m, "cannot create zip reader", err)
	}
	return extract(ctx, &zipWalker{zr: reader}, dst, c, m)
}

// zipWalker is a walker for zip files
type zipWalker struct {
	zr *zip.Reader
	fp int
}

// Type returns the file extension for zip files
func (z zipWalker) Type() string {
	return fileExtensionZIP
}

// Next returns the next entry in the zip archive
func (z *zipWalker) Next() (archiveEntry, error) {
	if z.fp >= len(z.zr.File) {
		return nil, io.EOF
	}
	defer func() { z.fp++ }()
	return &zipEntry{z.zr.File[z.fp]}, nil
}

// zipEntry is an entry in a zip archive
type zipEntry struct {
	zf *zip.File
}

// Name returns the name of the entry
func (z *zipEntry) Name() string {
	return z.zf.FileHeader.Name
}

// Size returns the size of the entry
func (z *zipEntry) Size() int64 {
	return int64(z.zf.FileHeader.UncompressedSize64)
}

// Mode returns the mode of the entry
func (z *zipEntry) Mode() os.FileMode {
	return z.zf.FileHeader.Mode()
}

// Linkname returns the linkname of the entry
func (z *zipEntry) Linkname() string {
	rc, _ := z.zf.Open()
	defer func() { rc.Close() }()
	data, _ := io.ReadAll(rc)
	return string(data)
}

// IsRegular returns true if the entry is a regular file
func (z *zipEntry) IsRegular() bool {
	return z.zf.FileHeader.Mode().Type() == 0
}

// IsDir returns true if the entry is a directory
func (z *zipEntry) IsDir() bool {
	return z.zf.FileHeader.Mode().Type() == os.ModeDir
}

// IsSymlink returns true if the entry is a symlink
func (z *zipEntry) IsSymlink() bool {
	return z.zf.FileHeader.Mode().Type() == os.ModeSymlink
}

// Open returns a reader for the entry
func (z *zipEntry) Open() (io.ReadCloser, error) {
	return z.zf.Open()
}

// Type returns the type of the entry
func (z *zipEntry) Type() fs.FileMode {
	return z.zf.FileHeader.Mode().Type()
}
