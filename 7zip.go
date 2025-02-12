// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/bodgit/sevenzip"
)

// fileExtension7zip is the file extension for 7zip files
const fileExtension7zip = "7z"

// magicBytes7zip are the magic bytes for 7zip files
var magicBytes7zip = [][]byte{
	{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
}

// is7zip checks if the header matches the magic bytes for 7zip files
func is7zip(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytes7zip)
}

// unpack7Zip sets a timeout for the ctx and starts the 7zip extraction from src to dst.
func unpack7Zip(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	// prepare telemetry data collection and emit
	td := &TelemetryData{ExtractedType: fileExtension7zip}
	defer cfg.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// check if src is a readerAt and an io.Seeker
	if sra, ok := src.(seekerReaderAt); ok {
		return process7zip(ctx, t, dst, sra, cfg, td)
	}

	// convert
	sra, err := readerToReaderAtSeeker(cfg, src)
	if err != nil {
		return handleError(cfg, td, "cannot convert reader to readerAt and seeker", err)
	}
	defer func() {
		if f, ok := sra.(*os.File); ok {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	return process7zip(ctx, t, dst, sra, cfg, td)
}

// process7zip checks ctx for cancellation, while it reads a 7zip file from src and extracts the contents to dst.
func process7zip(ctx context.Context, t Target, dst string, sra seekerReaderAt, cfg *Config, td *TelemetryData) error {
	// log extraction
	cfg.Logger().Info("extracting 7zip")

	// check if src is a seeker and readerAt
	s, _ := sra.(io.Seeker)
	ra, _ := sra.(io.ReaderAt)

	// get size of input and check if it exceeds maximum input size
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return handleError(cfg, td, "cannot seek to end of reader", err)
	}
	td.InputSize = size
	if cfg.MaxInputSize() != -1 && size > cfg.MaxInputSize() {
		return handleError(cfg, td, "cannot unarchive 7zip", fmt.Errorf("input size exceeds maximum input size"))
	}

	// create zip reader and extract
	reader, err := sevenzip.NewReader(ra, size)
	if err != nil {
		return handleError(cfg, td, "cannot create 7zip reader", err)
	}

	return extract(ctx, t, dst, &sevenZipWalker{reader, 0}, cfg, td)
}

// sevenZipWalker is a walker for 7zip files
type sevenZipWalker struct {
	r  *sevenzip.Reader
	fp int
}

// Type returns the file extension for 7zip files
func (z sevenZipWalker) Type() string {
	return fileExtension7zip
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
func (z *sevenZipEntry) Linkname() string {
	if !z.IsSymlink() {
		return ""
	}
	f, err := z.f.Open()
	if err != nil {
		return ""
	}
	defer f.Close()
	c, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	return string(c)
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
	return (z.f.FileInfo().Mode()&os.ModeSymlink != 0)
}

// Open returns a reader for the 7zip entry
func (z *sevenZipEntry) Open() (io.ReadCloser, error) {
	return z.f.Open()
}

// Type returns the type of the 7zip entry
func (z *sevenZipEntry) Type() fs.FileMode {
	return fs.FileMode(z.f.FileInfo().Mode())
}

// AccessTime returns the access time of the 7zip entry
func (z *sevenZipEntry) AccessTime() time.Time {
	return z.f.Accessed
}

// ModTime returns the modification time of the 7zip entry
func (z *sevenZipEntry) ModTime() time.Time {
	return z.f.Modified
}

// Sys returns the system information of the 7zip entry
func (z *sevenZipEntry) Sys() interface{} {
	return z.f.FileInfo().Sys()
}

// Gid is not supported for 7zip files. The used library does not provide
// this information. The function returns the group ID of the current process.
func (z *sevenZipEntry) Gid() int {
	return os.Getegid()
}

// Uid is not supported for 7zip files. The used library does not provide
// this information. The function returns the user ID of the current process.
func (z *sevenZipEntry) Uid() int {
	return os.Getuid()
}
