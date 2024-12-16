// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"context"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/nwaples/rardecode"
)

// fileExtensionRar is the file extension for Rar files.
const fileExtensionRar = "rar"

// magicBytesRar are the magic bytes for Rar files.
var magicBytesRar = [][]byte{
	{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00},       // Rar 1.5
	{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00}, // Rar 5.0
}

// isRar checks if the header matches the magic bytes for Rar files.
func isRar(data []byte) bool {
	return matchesMagicBytes(data, 0, magicBytesRar)
}

// unpackRar sets a timeout for the ctx and starts the Rar extraction from src to dst.
func unpackRar(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	// prepare telemetry data collection and emit
	td := &TelemetryData{ExtractedType: fileExtensionRar}
	defer cfg.TelemetryHook()(ctx, td)
	defer captureExtractionDuration(td, now())

	// cache reader if needed
	reader, err := readerToReaderAtSeeker(cfg, src)
	if err != nil {
		return handleError(cfg, td, "cannot cache reader", err)
	}

	return processRar(ctx, t, dst, reader.(io.Reader), cfg, td)
}

// processRar extracts a Rar archive from src to dst.
func processRar(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config, td *TelemetryData) error {
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

// rarWalker is an archiveWalker for Rar files.
type rarWalker struct {
	r *rardecode.Reader
}

// Type returns the file extension for rar files.
func (rw *rarWalker) Type() string {
	return fileExtensionRar
}

// Next returns the next entry in the rar file.
func (rw *rarWalker) Next() (archiveEntry, error) {
	fh, err := rw.r.Next()
	if err != nil {
		return nil, err
	}
	re := &rarEntry{fh, rw.r}
	return re, nil
}

// rarEntry is an archiveEntry for Rar files.
type rarEntry struct {
	f *rardecode.FileHeader
	r io.Reader
}

// Name returns the name of the file.
func (r *rarEntry) Name() string {
	return r.f.Name
}

// Size returns the size of the file.
func (r *rarEntry) Size() int64 {
	return r.f.UnPackedSize
}

// Mode returns the mode of the file.
func (r *rarEntry) Mode() os.FileMode {
	return r.f.Mode()
}

// Linkname symlinks are not supported.
func (r *rarEntry) Linkname() string {
	return ""
}

// IsRegular returns true if the file is a regular file.
func (r *rarEntry) IsRegular() bool {
	return r.f.Mode().IsRegular()
}

// IsDir returns true if the file is a directory.
func (r *rarEntry) IsDir() bool {
	return r.f.IsDir
}

// IsSymlink returns true if the file is a symlink.
func (r *rarEntry) IsSymlink() bool {
	return false
}

// Type returns the type of the file.
func (r *rarEntry) Type() fs.FileMode {
	return r.f.Mode().Type()
}

// Open returns a reader for the file.
func (r *rarEntry) Open() (io.ReadCloser, error) {
	return io.NopCloser(r.r), nil
}

// AccessTime returns the access time of the file.
func (r *rarEntry) AccessTime() time.Time {
	return r.f.AccessTime
}

// ModTime returns the modification time of the file.
func (r *rarEntry) ModTime() time.Time {
	return r.f.ModificationTime
}

// Sys returns the system information of the file.
func (r *rarEntry) Sys() interface{} {
	return r.f
}

// Gid is not supported for Rar files. The used library does not provide
// this information. The function returns the group ID of the current process.
func (r *rarEntry) Gid() int {
	return 0
}

// Uid is not supported for Rar files. The used library does not provide
// this information. The function returns the user ID of the current process.
func (r *rarEntry) Uid() int {
	return 0
}
