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

// FileExtension7zip is the file extension for 7zip files
const FileExtension7zip = "7z"

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
	td := &telemetry.Data{ExtractedType: FileExtension7zip}
	defer c.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// check if src is a readerAt and an io.Seeker
	if sra, ok := src.(SeekerReaderAt); ok {
		return unpack7zip(ctx, sra, dst, c, td)
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

	return unpack7zip(ctx, sra, dst, c, td)
}

// unpack7zip checks ctx for cancellation, while it reads a 7zip file from src and extracts the contents to dst.
func unpack7zip(ctx context.Context, src SeekerReaderAt, dst string, c *config.Config, m *telemetry.Data) error {

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

// sevenZipWalker is a walker for 7zip files
type sevenZipWalker struct {
	r  *sevenzip.Reader
	fp int
}

// Type returns the file extension for 7zip files
func (z sevenZipWalker) Type() string {
	return FileExtension7zip
}

// Next returns the next entry in the 7zip file
func (z *sevenZipWalker) Next() (archiveEntry, error) {
	if z.fp >= len(z.r.File) {
		return nil, io.EOF
	}
	defer func() { z.fp++ }()
	return &sevenZipEntry{z.r.File[z.fp]}, nil
}

// sevenZipEntry is an entry in a 7zip file
type sevenZipEntry struct {
	f *sevenzip.File
}

// Name returns the name of the 7zip entry
func (z *sevenZipEntry) Name() string {
	return z.f.Name
}

// Size returns the size of the 7zip entry
func (z *sevenZipEntry) Size() int64 {
	return int64(z.f.FileInfo().Size())
}

// Mode returns the mode of the 7zip entry
func (z *sevenZipEntry) Mode() os.FileMode {
	return z.f.FileInfo().Mode()
}

// Linkname returns the linkname of the 7zip entry
// Remark: 7zip does not support symlinks
func (z *sevenZipEntry) Linkname() string {
	return ""
}

// IsRegular returns true if the 7zip entry is a regular file
func (z *sevenZipEntry) IsRegular() bool {
	return z.f.FileInfo().Mode().IsRegular()
}

// IsDir returns true if the 7zip entry is a directory
func (z *sevenZipEntry) IsDir() bool {
	return z.f.FileInfo().Mode().IsDir()
}

// IsSymlink returns true if the 7zip entry is a symlink
// Remark: 7zip does not support symlinks
func (z *sevenZipEntry) IsSymlink() bool {
	return false
}

// Open returns a reader for the 7zip entry
func (z *sevenZipEntry) Open() (io.ReadCloser, error) {
	return z.f.Open()
}

// Type returns the type of the 7zip entry
func (z *sevenZipEntry) Type() fs.FileMode {
	return fs.FileMode(z.f.FileInfo().Mode())
}
