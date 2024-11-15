// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// now is a function point that returns time.Now to the caller.
var now = time.Now

// seekerReaderAt is a struct that combines the io.ReaderAt and io.Seeker interfaces
type seekerReaderAt interface {
	io.ReaderAt
	io.Seeker
}

// checkPatterns checks if the given path matches any of the given patterns.
// If no patterns are given, the function returns true.
func checkPatterns(patterns []string, path string) (bool, error) {
	// no patterns given
	if len(patterns) == 0 {
		return true, nil
	}

	// check if path matches any pattern
	for _, pattern := range patterns {
		if match, err := filepath.Match(pattern, path); err != nil {
			return false, fmt.Errorf("failed to match pattern: %w", err)
		} else if match {
			return true, nil
		}
	}
	return false, nil
}

// captureExtractionDuration captures the duration of the extraction
func captureExtractionDuration(m *TelemetryData, start time.Time) {
	stop := now()
	m.ExtractionDuration = stop.Sub(start)
}

// captureInputSize captures the input size of the extraction
func captureInputSize(td *TelemetryData, ler *limitErrorReader) {
	td.InputSize = int64(ler.ReadBytes())
}

// unpackFunc is a function that extracts the contents from src and extracts them to dst.
type unpackFunc func(context.Context, Target, string, io.Reader, *Config) error

// headerCheckFunc is a function that checks if the given header matches the expected magic bytes.
type headerCheckFunc func([]byte) bool

type extractor struct {
	Unpacker    unpackFunc
	HeaderCheck headerCheckFunc
	MagicBytes  [][]byte
	Offset      int
}

type extractors map[string]extractor

// getUnpackFunction identifies the correct extractor based on magic bytes.
func (e extractors) getUnpackFunction(data []byte) unpackFunc {
	// find extractor with longest suffix match
	for _, ex := range e {
		if ex.HeaderCheck(data) {
			return ex.Unpacker
		}
	}

	// no matching reader found
	return nil
}

// getUnpackFunctionByFileName identifies the correct extractor based on file extension.
func (e extractors) getUnpackFunctionByFileName(ext string) unpackFunc {
	// get file extension from file name
	ext = strings.ToLower(ext)
	if strings.Contains(ext, ".") {
		ext = filepath.Ext(ext)
		ext = strings.Replace(ext, ".", "", -1) // remove leading dot if the file extension is the only part of the file name (e.g. ".tar")
	}

	if ae, found := e[ext]; found {
		return ae.Unpacker
	}

	// no matching reader found
	return nil
}

// GetUnpackFunction identifies the correct extractor based on magic bytes.
func (e extractors) GetUnpackFunction(header []byte, ext string) unpackFunc {
	// find extractor by header
	if unpacker := e.getUnpackFunction(header); unpacker != nil {
		return unpacker
	}

	// find extractor by file extension
	if unpacker := e.getUnpackFunctionByFileName(ext); unpacker != nil {
		return unpacker
	}

	// no matching reader found
	return nil
}

// Extensions returns a string with all available file extensions.
func (e extractors) Extensions() string {
	var types []string
	for t := range e {
		types = append(types, t)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}

// AvailableExtractors is collection of new extractor functions with
// the required magic bytes and potential offset
var AvailableExtractors = extractors{
	fileExtension7zip: {
		Unpacker:    unpack7Zip,
		HeaderCheck: is7zip,
		MagicBytes:  magicBytes7zip,
	},
	fileExtensionBrotli: {
		Unpacker:    unpackBrotli,
		HeaderCheck: isBrotli,
	},
	fileExtensionBzip2: {
		Unpacker:    unpackBzip2,
		HeaderCheck: isBzip2,
		MagicBytes:  magicBytesBzip2,
	},
	fileExtensionGZip: {
		Unpacker:    unpackGZip,
		HeaderCheck: isGZip,
		MagicBytes:  magicBytesGZip,
	},
	fileExtensionLZ4: {
		Unpacker:    unpackLZ4,
		HeaderCheck: isLZ4,
		MagicBytes:  magicBytesLZ4,
	},
	fileExtensionSnappy: {
		Unpacker:    unpackSnappy,
		HeaderCheck: isSnappy,
		MagicBytes:  magicBytesSnappy,
	},
	fileExtensionTar: {
		Unpacker:    unpackTar,
		HeaderCheck: isTar,
		MagicBytes:  magicBytesTar,
		Offset:      offsetTar,
	},
	fileExtensionTarGZip: {
		Unpacker:    unpackGZip,
		HeaderCheck: isGZip,
		MagicBytes:  magicBytesGZip,
	},
	fileExtensionXz: {
		Unpacker:    unpackXz,
		HeaderCheck: isXz,
		MagicBytes:  magicBytesXz,
	},
	fileExtensionZIP: {
		Unpacker:    unpackZip,
		HeaderCheck: isZip,
		MagicBytes:  magicBytesZIP,
	},
	fileExtensionZlib: {
		Unpacker:    unpackZlib,
		HeaderCheck: isZlib,
		MagicBytes:  magicBytesZlib,
	},
	fileExtensionZstd: {
		Unpacker:    unpackZstd,
		HeaderCheck: isZstd,
		MagicBytes:  magicBytesZstd,
	},
	fileExtensionRar: {
		Unpacker:    unpackRar,
		HeaderCheck: isRar,
		MagicBytes:  magicBytesRar,
	},
}

var maxHeaderLength int

// init calculates the maximum header length
func init() {
	for _, ex := range AvailableExtractors {
		needs := ex.Offset
		for _, mb := range ex.MagicBytes {
			if len(mb)+ex.Offset > needs {
				needs = len(mb) + ex.Offset
			}
		}
		if needs > maxHeaderLength {
			maxHeaderLength = needs
		}
	}
}

func matchesMagicBytes(data []byte, offset int, magicBytes [][]byte) bool {
	// check all possible magic bytes until match is found
	for _, mb := range magicBytes {
		// check if header is long enough
		if offset+len(mb) > len(data) {
			continue
		}

		// check for byte match
		if bytes.Equal(mb, data[offset:offset+len(mb)]) {
			return true
		}
	}

	// no match found
	return false
}

// handleError increases the error counter, sets the latest error and
// decides if extraction should continue.
func handleError(cfg *Config, td *TelemetryData, msg string, err error) error {
	// check if error is an unsupported file
	if uf, ok := err.(*UnsupportedFileError); ok {

		// increase unsupported file counter and set last unsupported file to unknown
		td.UnsupportedFiles++
		td.LastUnsupportedFile = uf.Filename()

		// log error and return nil
		if cfg.ContinueOnUnsupportedFiles() {
			cfg.Logger().Error("not supported file", "msg", msg, "error", err)
			return nil
		}
	}

	// increase error counter and set error
	td.ExtractionErrors++
	td.LastExtractionError = fmt.Errorf("%s: %w", msg, err)

	// do not end on error
	if cfg.ContinueOnError() {
		cfg.Logger().Error(msg, "error", err)
		return nil
	}

	// end extraction on error
	return td.LastExtractionError
}

// extract checks ctx for cancellation, while it reads a tar file from src and extracts the contents to dst.
func extract(ctx context.Context, t Target, dst string, src archiveWalker, cfg *Config, td *TelemetryData) error {

	// start extraction
	cfg.Logger().Info("start extraction", "type", src.Type())
	var fileCounter int64
	var extractionSize int64

	for {
		// check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// get next file
		ae, err := src.Next()

		switch {

		// if no more files are found exit loop
		case err == io.EOF:
			// extraction finished
			return nil

		// handle other errors and end extraction or continue
		case err != nil:
			if err := handleError(cfg, td, "error reading", err); err != nil {
				return err
			}
			continue

		// if the header is nil, just skip it (not sure how this happens)
		case ae == nil:
			continue
		}

		// check for to many files (including folder and symlinks) in archive
		fileCounter++

		// check if maximum of files (including folder and symlinks) is exceeded
		if err := cfg.CheckMaxFiles(fileCounter); err != nil {
			return handleError(cfg, td, "max objects check failed", err)
		}

		// check if file needs to match patterns
		match, err := checkPatterns(cfg.Patterns(), ae.Name())
		if err != nil {
			return handleError(cfg, td, "cannot check pattern", err)
		}
		if !match {
			cfg.Logger().Info("skipping file (pattern mismatch)", "name", ae.Name())
			td.PatternMismatches++
			continue
		}

		cfg.Logger().Debug("extract", "name", ae.Name())
		switch {

		// if its a dir and it doesn't exist create it
		case ae.IsDir():

			// handle directory
			if err := createDir(t, dst, ae.Name(), ae.Mode(), cfg); err != nil {
				if err := handleError(cfg, td, "failed to create safe directory", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store telemetry and continue
			td.ExtractedDirs++
			continue

		// if it's a file create it
		case ae.IsRegular():

			// check extraction size forecast
			if err := cfg.CheckExtractionSize(extractionSize + ae.Size()); err != nil {
				return handleError(cfg, td, "max extraction size exceeded", err)
			}

			// open file inm archive
			fin, err := ae.Open()
			if err != nil {
				return handleError(cfg, td, "failed to open file", err)
			}
			defer fin.Close()

			// create file
			n, err := createFile(t, dst, ae.Name(), fin, ae.Mode(), cfg.MaxExtractionSize()-extractionSize, cfg)
			extractionSize = extractionSize + n
			td.ExtractionSize = extractionSize
			if err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(cfg, td, "failed to create safe file", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store telemetry
			td.ExtractedFiles++

			continue

		// its a symlink !!
		case ae.IsSymlink():

			// check if symlinks are allowed
			if cfg.DenySymlinkExtraction() {

				err := unsupportedFile(ae.Name())
				if err := handleError(cfg, td, "symlink extraction disabled", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// create link
			if err := createSymlink(t, dst, ae.Name(), ae.Linkname(), cfg); err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(cfg, td, "failed to create safe symlink", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store telemetry and continue
			td.ExtractedSymlinks++
			continue

		default:

			// tar specific: check for git comment file `pax_global_header` from type `67` and skip
			if ae.Type()&tar.TypeXGlobalHeader == tar.TypeXGlobalHeader && ae.Name() == "pax_global_header" {
				continue
			}

			err := unsupportedFile(ae.Name())
			msg := fmt.Sprintf("unsupported filetype in archive (%x)", ae.Mode())
			if err := handleError(cfg, td, msg, err); err != nil {
				return err
			}

			// do not end on error
			continue
		}
	}
}

// readerToReaderAtSeeker converts an io.Reader to an io.ReaderAt and io.Seeker
func readerToReaderAtSeeker(c *Config, r io.Reader) (seekerReaderAt, error) {
	if s, ok := r.(seekerReaderAt); ok {
		return s, nil
	}

	// check if reader is a file
	if f, ok := r.(*os.File); ok {
		return f, nil
	}

	// check if reader is a buffer
	if b, ok := r.(*bytes.Buffer); ok {
		return bytes.NewReader(b.Bytes()), nil
	}

	// limit reader
	ler := newLimitErrorReader(r, c.MaxInputSize())

	// check how to cache
	if c.CacheInMemory() {
		b, err := io.ReadAll(ler)
		if err != nil {
			return nil, fmt.Errorf("cannot read all from reader: %w", err)
		}
		return bytes.NewReader(b), nil
	}

	// create temp file
	tmpFile, err := os.CreateTemp("", "extractor-*")
	if err != nil {
		return nil, err
	}

	// copy reader to temp file
	if _, err := io.Copy(tmpFile, ler); err != nil {
		defer os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("cannot copy reader to file: %w", err)
	}

	// seek to start
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		defer os.Remove(tmpFile.Name())
		return nil, err
	}

	// return temp file
	return tmpFile, nil
}

// ErrUnsupportedFile is an error that indicates that the file is not supported.
var ErrUnsupportedFile = errors.New("unsupported file")

// unsupportedFile returns an error that indicates that the file is not supported.
func unsupportedFile(filename string) error {
	return &UnsupportedFileError{error: ErrUnsupportedFile, filename: filename}
}

// UnsupportedFileError is an error that indicates that the file is not supported.
type UnsupportedFileError struct {
	error
	filename string
}

// Filename returns the filename of the unsupported file.
func (e *UnsupportedFileError) Filename() string {
	return e.filename
}

// Unwrap returns the underlying error.
func (e *UnsupportedFileError) Unwrap() error {
	return e.error
}

// Error returns the error message.
func (e UnsupportedFileError) Error() string {
	return fmt.Sprintf("%v: %s", e.error, e.filename)
}

// getHeader reads the header from src and returns it. If src is a io.Seeker, the header is read
// directly from the reader and the reader gets reset. If src is not a io.Seeker, the header is read
// and transformed into a HeaderReader, which is returned as the second return value. If an error
// occurs, the header is nil and the error is returned as the third return value
func getHeader(src io.Reader) ([]byte, io.Reader, error) {
	// check if source offers seek and preserve type of source
	if s, ok := src.(io.Seeker); ok {

		// allocate buffer for header
		header := make([]byte, maxHeaderLength)

		// read header from source
		_, err := src.Read(header)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read header: %w", err)
		}
		// reset reader
		_, err = s.Seek(0, io.SeekStart)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to reset reader: %w", err)
		}
		return header, src, nil
	}

	headerReader, err := newHeaderReader(src, maxHeaderLength)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create header reader: %w", err)
	}
	return headerReader.PeekHeader(), headerReader, nil
}
