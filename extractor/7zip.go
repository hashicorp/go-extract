package extractor

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/bodgit/sevenzip"
	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

// fileExtension7zip is the file extension for 7zip files
var fileExtension7zip = "7z"

// magicBytes7zip are the magic bytes for 7zip files
var magicBytes7zip = [][]byte{
	{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
}

// Is7zip checks if the header matches the magic bytes for 7zip files
func Is7zip(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytes7zip)
}

// Unpack7Zip sets a timeout for the ctx and starts the 7zip extraction from src to dst.
func Unpack7Zip(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// prepare telemetry data collection and emit
	td := &telemetry.Data{ExtractedType: fileExtension7zip}
	defer c.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// check if src is a readerAt and an io.Seeker
	if sra, ok := src.(SeekerReaderAt); ok {
		return unpack7zipReaderAtSeeker(ctx, sra, dst, c, td)
	}

	// convert
	sra, err := ReaderToReaderAtSeeker(c, src)
	if err != nil {
		return handleError(c, td, "cannot convert reader to readerAt and seeker", err)
	}
	defer func() {
		if f, ok := sra.(*os.File); ok {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	return unpack7zipReaderAtSeeker(ctx, sra, dst, c, td)
}

func unpack7zipReaderAtSeeker(ctx context.Context, src SeekerReaderAt, dst string, c *config.Config, m *telemetry.Data) error {

	// log extraction
	c.Logger().Info("extracting 7zip")

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
		return handleError(c, m, "cannot unarchive 7zip", fmt.Errorf("input size exceeds maximum input size"))
	}

	// create zip reader and extract
	reader, err := sevenzip.NewReader(ra, size)
	if err != nil {
		return handleError(c, m, "cannot create 7zip reader", err)
	}

	return extract(ctx, &sevenZipWalker{reader, 0}, dst, c, m)
}

type sevenZipWalker struct {
	r  *sevenzip.Reader
	fp int
}

func (z sevenZipWalker) Type() string {
	return fileExtension7zip
}

func (z *sevenZipWalker) Next() (archiveEntry, error) {
	if z.fp >= len(z.r.File) {
		return nil, io.EOF
	}
	defer func() { z.fp++ }()
	return &sevenZipEntry{z.r.File[z.fp]}, nil
}

type sevenZipEntry struct {
	f *sevenzip.File
}

func (z *sevenZipEntry) Name() string {
	return z.f.Name
}

func (z *sevenZipEntry) Size() int64 {
	return int64(z.f.FileInfo().Size())
}

func (z *sevenZipEntry) Mode() os.FileMode {
	return z.f.FileInfo().Mode()
}

func (z *sevenZipEntry) Linkname() string {
	return ""
}

func (z *sevenZipEntry) IsRegular() bool {
	return z.f.FileInfo().Mode().IsRegular()
}

func (z *sevenZipEntry) IsDir() bool {
	return z.f.FileInfo().Mode().IsDir()
}

func (z *sevenZipEntry) IsSymlink() bool {
	return false
}

func (z *sevenZipEntry) Open() (io.ReadCloser, error) {
	return z.f.Open()
}

func (z *sevenZipEntry) Type() fs.FileMode {
	return fs.FileMode(z.f.FileInfo().Mode())
}
