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
var fileExtensionZIP = "zip"

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
		return unpackZipReaderAtSeeker(ctx, sra, dst, c, td)
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

	return unpackZipReaderAtSeeker(ctx, sra, dst, c, td)
}

// unpackZipReaderAtSeeker checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
// src is a readerAt and a seeker. If the InputSize exceeds the maximum input size, the function returns an error.
func unpackZipReaderAtSeeker(ctx context.Context, src SeekerReaderAt, dst string, c *config.Config, m *telemetry.Data) error {

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

type zipWalker struct {
	zr *zip.Reader
	fp int
}

func (z zipWalker) Type() string {
	return fileExtensionZIP
}

func (z *zipWalker) Next() (archiveEntry, error) {
	if z.fp >= len(z.zr.File) {
		return nil, io.EOF
	}
	defer func() { z.fp++ }()
	return &zipEntry{z.zr.File[z.fp]}, nil
}

type zipEntry struct {
	zf *zip.File
}

func (z *zipEntry) Name() string {
	return z.zf.FileHeader.Name
}

func (z *zipEntry) Size() int64 {
	return int64(z.zf.FileHeader.UncompressedSize64)
}

func (z *zipEntry) Mode() os.FileMode {
	return z.zf.FileHeader.Mode()
}

func (z *zipEntry) Linkname() string {
	rc, _ := z.zf.Open()
	defer func() { rc.Close() }()
	data, _ := io.ReadAll(rc)
	return string(data)
}

func (z *zipEntry) IsRegular() bool {
	return z.zf.FileHeader.Mode().Type() == 0
}

func (z *zipEntry) IsDir() bool {
	return z.zf.FileHeader.Mode().Type() == os.ModeDir
}

func (z *zipEntry) IsSymlink() bool {
	return z.zf.FileHeader.Mode().Type() == os.ModeSymlink
}

func (z *zipEntry) Open() (io.ReadCloser, error) {
	return z.zf.Open()
}

func (z *zipEntry) Type() fs.FileMode {
	return z.zf.FileHeader.Mode().Type()
}
