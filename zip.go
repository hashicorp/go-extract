// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// fileExtensionZip is the file extension for zip files.
const fileExtensionZip = "zip"

// magicBytesZip contains the magic bytes for a zip archive.
// reference: https://golang.org/pkg/archive/zip/
var magicBytesZip = [][]byte{
	{0x50, 0x4B, 0x03, 0x04},
}

// isZip checks if data is a zip archive. It returns true if data is a zip archive and false if data is not a zip archive.
func isZip(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytesZip)
}

// Unpack sets a timeout for the ctx and starts the zip extraction from src to dst. It returns an error if the extraction failed.
func unpackZip(ctx context.Context, t Target, dst string, src io.Reader, c *Config) error {
	// prepare telemetry data collection and emit
	td := &TelemetryData{ExtractedType: fileExtensionZip}
	defer c.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// check if src is a readerAt and an io.Seeker
	if sra, ok := src.(seekerReaderAt); ok {
		return processZip(ctx, t, sra, dst, c, td)
	}

	// convert
	sra, err := readerToReaderAtSeeker(c, src)
	if err != nil {
		return handleError(c, td, "cannot convert reader to readerAt and seeker", err)
	}
	defer func() {
		if f, ok := src.(*os.File); ok {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	return processZip(ctx, t, sra, dst, c, td)
}

// processZip checks ctx for cancellation, while it reads a zip file from src and extracts the contents to dst.
// src is a readerAt and a seeker. If the InputSize exceeds the maximum input size, the function returns an error.
func processZip(ctx context.Context, t Target, src seekerReaderAt, dst string, cfg *Config, m *TelemetryData) error {

	// log extraction
	cfg.Logger().Info("extracting zip")

	// check if src is a seeker and readerAt
	s, _ := src.(io.Seeker)
	ra, _ := src.(io.ReaderAt)

	// get size of input and check if it exceeds maximum input size
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return handleError(cfg, m, "cannot seek to end of reader", err)
	}
	m.InputSize = size
	if cfg.MaxInputSize() != -1 && size > cfg.MaxInputSize() {
		return handleError(cfg, m, "cannot unpack zip", fmt.Errorf("input size exceeds maximum input size"))
	}

	// create zip reader and extract
	reader, err := zip.NewReader(ra, size)
	if err != nil {
		return handleError(cfg, m, "cannot create zip reader", err)
	}
	return extract(ctx, t, dst, &zipWalker{zr: reader}, cfg, m)
}

// zipWalker is a walker for zip files
type zipWalker struct {
	zr *zip.Reader
	fp int
}

// Type returns the file extension for zip files
func (z zipWalker) Type() string {
	return fileExtensionZip
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

// AccessTime returns the access time of the entry
func (z *zipEntry) AccessTime() time.Time {
	return z.zf.FileHeader.FileInfo().ModTime()
}

// ModTime returns the modification time of the entry
func (z *zipEntry) ModTime() time.Time {
	return z.zf.FileHeader.FileInfo().ModTime()
}

// Sys returns the system information of the entry
func (z *zipEntry) Sys() interface{} {
	return z.zf.FileHeader
}

// Gid returns the group ID of the entry
func (z *zipEntry) Gid() int {
	return os.Getgid()
}

// Uid returns the user ID of the entry
func (z *zipEntry) Uid() int {
	return os.Getuid()
}
