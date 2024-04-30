package extractor

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
	"github.com/hashicorp/go-extract/telemetry"
)

// now is a function point that returns time.Now to the caller.
var now = time.Now

// unpackTarget is the target that is used for extraction
var unpackTarget target.Target

// SeekerReaderAt is a struct that combines the io.ReaderAt and io.Seeker interfaces
type SeekerReaderAt interface {
	io.ReaderAt
	io.Seeker
}

// determineOutputName determines the output name and directory for the extracted content
func determineOutputName(dst string, src io.Reader) (string, string) {

	// check if dst is specified and not a directory
	if dst != "." && dst != "" {
		if stat, err := os.Stat(dst); os.IsNotExist(err) || stat.Mode()&fs.ModeDir == 0 {
			return filepath.Dir(dst), filepath.Base(dst)
		}
	}

	// check if src is a file and the filename is ending with the suffix
	// remove the suffix from the filename and use it as output name
	if f, ok := src.(*os.File); ok {

		name := filepath.Base(f.Name())
		name = strings.TrimSuffix(name, ".")
		newName := strings.TrimSuffix(name, filepath.Ext(name))

		// check if the filename is valid
		if err := validFilename(newName); err != nil {
			return dst, "goextract-decompressed-content"
		}

		if name != newName {
			return dst, newName
		}

		// if the filename is not ending with the suffix, use the suffix as output name
		return dst, fmt.Sprintf("%s.decompressed", newName)
	}

	return dst, "goextract-decompressed-content"
}

// validFilename checks if the given filename is a valid filename on
// the operating system
func validFilename(name string) error {

	// trim leading slash and backslash
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimPrefix(name, "\\")

	// basic filename check
	if !filepath.IsLocal(name) {
		return fmt.Errorf("invalid filename: %s", name)
	}

	if strings.HasSuffix(name, "/") {
		return fmt.Errorf("trailing slash (%s)", name)
	}

	if name == "." {
		return fmt.Errorf("reserved name: %s", name)
	}

	// no issues found
	return nil
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
			return false, fmt.Errorf("failed to match pattern: %s", err)
		} else if match {
			return true, nil
		}
	}
	return false, nil
}

// captureExtractionDuration captures the duration of the extraction
func captureExtractionDuration(m *telemetry.Data, start time.Time) {
	stop := now()
	m.ExtractionDuration = stop.Sub(start)
}

// captureInputSize captures the input size of the extraction
func captureInputSize(td *telemetry.Data, ler *LimitErrorReader) {
	td.InputSize = int64(ler.ReadBytes())
}

// UnpackFunc is a function that extracts the contents from src and extracts them to dst.
type UnpackFunc func(context.Context, io.Reader, string, *config.Config) error

// HeaderCheck is a function that checks if the given header matches the expected magic bytes.
type HeaderCheck func([]byte) bool

type AvailableExtractor struct {
	Unpacker    UnpackFunc
	HeaderCheck HeaderCheck
	MagicBytes  [][]byte
	Offset      int
}

// AvailableExtractors is collection of new extractor functions with
// the required magic bytes and potential offset
var AvailableExtractors = map[string]AvailableExtractor{
	FileExtension7zip: {
		Unpacker:    Unpack7Zip,
		HeaderCheck: Is7zip,
		MagicBytes:  magicBytes7zip,
	},
	FileExtensionBrotli: {
		Unpacker:    UnpackBrotli,
		HeaderCheck: IsBrotli,
	},
	FileExtensionBzip2: {
		Unpacker:    UnpackBzip2,
		HeaderCheck: IsBzip2,
		MagicBytes:  magicBytesBzip2,
	},
	FileExtensionGZip: {
		Unpacker:    UnpackGZip,
		HeaderCheck: IsGZip,
		MagicBytes:  magicBytesGZip,
	},
	FileExtensionLZ4: {
		Unpacker:    UnpackLZ4,
		HeaderCheck: IsLZ4,
		MagicBytes:  magicBytesLZ4,
	},
	FileExtensionSnappy: {
		Unpacker:    UnpackSnappy,
		HeaderCheck: IsSnappy,
		MagicBytes:  magicBytesSnappy,
	},
	FileExtensionTar: {
		Unpacker:    UnpackTar,
		HeaderCheck: IsTar,
		MagicBytes:  magicBytesTar,
		Offset:      offsetTar,
	},
	FileExtensionTarGZip: {
		Unpacker:    UnpackGZip,
		HeaderCheck: IsGZip,
		MagicBytes:  magicBytesGZip,
	},
	FileExtensionXz: {
		Unpacker:    UnpackXz,
		HeaderCheck: IsXz,
		MagicBytes:  magicBytesXz,
	},
	FileExtensionZIP: {
		Unpacker:    UnpackZip,
		HeaderCheck: IsZip,
		MagicBytes:  magicBytesZIP,
	},
	FileExtensionZlib: {
		Unpacker:    UnpackZlib,
		HeaderCheck: IsZlib,
		MagicBytes:  magicBytesZlib,
	},
	FileExtensionZstd: {
		Unpacker:    UnpackZstd,
		HeaderCheck: IsZstd,
		MagicBytes:  magicBytesZstd,
	},
}

var MaxHeaderLength int

// init calculates the maximum header length
func init() {
	for _, ex := range AvailableExtractors {
		needs := ex.Offset
		for _, mb := range ex.MagicBytes {
			if len(mb)+ex.Offset > needs {
				needs = len(mb) + ex.Offset
			}
		}
		if needs > MaxHeaderLength {
			MaxHeaderLength = needs
		}
	}

	// set default target
	unpackTarget = target.NewOS()
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
func handleError(c *config.Config, td *telemetry.Data, msg string, err error) error {

	// increase error counter and set error
	td.ExtractionErrors++
	td.LastExtractionError = fmt.Errorf("%s: %s", msg, err)

	// do not end on error
	if c.ContinueOnError() {
		c.Logger().Error(msg, "error", err)
		return nil
	}

	// end extraction on error
	return td.LastExtractionError
}

// extract checks ctx for cancellation, while it reads a tar file from src and extracts the contents to dst.
func extract(ctx context.Context, src archiveWalker, dst string, c *config.Config, td *telemetry.Data) error {

	// start extraction
	c.Logger().Info("start extraction", "type", src.Type())
	var objectCounter int64
	var extractionSize uint64

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

		// return any other error
		case err != nil:
			return handleError(c, td, "error reading", err)

		// if the header is nil, just skip it (not sure how this happens)
		case ae == nil:
			continue
		}

		// check for to many objects in archive
		objectCounter++

		// check if maximum of objects is exceeded
		if err := c.CheckMaxObjects(objectCounter); err != nil {
			return handleError(c, td, "max objects check failed", err)
		}

		// check if file needs to match patterns
		match, err := checkPatterns(c.Patterns(), ae.Name())
		if err != nil {
			return handleError(c, td, "cannot check pattern", err)
		}
		if !match {
			c.Logger().Info("skipping file (pattern mismatch)", "name", ae.Name())
			td.PatternMismatches++
			continue
		}

		c.Logger().Debug("extract", "name", ae.Name())
		switch {

		// if its a dir and it doesn't exist create it
		case ae.IsDir():

			// handle directory
			if err := unpackTarget.CreateSafeDir(c, dst, ae.Name(), ae.Mode()); err != nil {
				if err := handleError(c, td, "failed to create safe directory", err); err != nil {
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

			// check extraction size
			extractionSize = extractionSize + uint64(ae.Size())
			if err := c.CheckExtractionSize(int64(extractionSize)); err != nil {
				return handleError(c, td, "max extraction size exceeded", err)
			}

			// open file inm archive
			fin, err := ae.Open()
			if err != nil {
				return handleError(c, td, "failed to open file", err)
			}
			defer fin.Close()

			// create file
			if err := unpackTarget.CreateSafeFile(c, dst, ae.Name(), fin, ae.Mode()); err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(c, td, "failed to create safe file", err); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// store telemetry
			td.ExtractionSize = int64(extractionSize)
			td.ExtractedFiles++
			continue

		// its a symlink !!
		case ae.IsSymlink():

			// check if symlinks are allowed
			if c.DenySymlinkExtraction() {

				// check for continue for unsupported files
				if c.ContinueOnUnsupportedFiles() {
					td.UnsupportedFiles++
					td.LastUnsupportedFile = ae.Name()
					continue
				}

				if err := handleError(c, td, "symlinks are not allowed", fmt.Errorf("symlinks are not allowed")); err != nil {
					return err
				}

				// do not end on error
				continue
			}

			// create link
			if err := unpackTarget.CreateSafeSymlink(c, dst, ae.Name(), ae.Linkname()); err != nil {

				// increase error counter, set error and end if necessary
				if err := handleError(c, td, "failed to create safe symlink", err); err != nil {
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

			// check if unsupported files should be skipped
			if c.ContinueOnUnsupportedFiles() {
				td.UnsupportedFiles++
				td.LastUnsupportedFile = ae.Name()
				continue
			}

			// increase error counter, set error and end if necessary
			if err := handleError(c, td, "cannot extract file", fmt.Errorf("unsupported filetype in archive (%x)", ae.Mode())); err != nil {
				return err
			}

			// do not end on error
			continue
		}
	}
}

// ReaderToReaderAtSeeker converts an io.Reader to an io.ReaderAt and io.Seeker
func ReaderToReaderAtSeeker(c *config.Config, r io.Reader) (SeekerReaderAt, error) {

	if s, ok := r.(SeekerReaderAt); ok {
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
	ler := NewLimitErrorReader(r, c.MaxInputSize())

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
