package extract

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	// ErrNoExtractorFound is returned when no extractor is found for the given file type.
	ErrNoExtractorFound = fmt.Errorf("extract: no extractor found for file type")

	// ErrUnsupportedFileType is returned when the file type is not supported.
	ErrUnsupportedFileType = fmt.Errorf("extract: unsupported file type")

	// ErrFailedToReadHeader is returned when the header of the file cannot be read.
	ErrFailedToReadHeader = fmt.Errorf("extract: failed to read header")

	// ErrFailedToExtract is returned when the file cannot be extracted.
	ErrFailedToUnpack = fmt.Errorf("extract: failed to unpack")
)

// // Target is an interface that defines the methods that a target must implement
// // so that the unpacking process can be done.
// type Target Target

// NewMemoryTarget returns a new memory target that provides an in-memory filesystem (that implements [io/fs.FS]).
func NewMemoryTarget() Target {
	return NewMemory()
}

// NewDiskTarget returns a new OS target that uses the filesystem of the operating system.
func NewDiskTarget() Target {
	return NewDisk()
}

// Unpack unpacks the given source to the destination, according to the given configuration,
// using the default OS  If cfg is nil, the default configuration
// is used for extraction. If an error occurs, it is returned.
func Unpack(ctx context.Context, src io.Reader, dst string, cfg *Config) error {
	return UnpackTo(ctx, NewDisk(), dst, src, cfg)
}

// UnpackTo unpacks the given source to the destination, according to the given configuration,
// using the given [Target]. If cfg is nil, the default configuration is used for extraction.
// If an error occurs, it is returned.
func UnpackTo(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	if cfg == nil {
		cfg = NewConfig()
	}
	if et := cfg.ExtractType(); len(et) > 0 {
		if ae, found := AvailableExtractors[et]; found {
			if et == FileExtensionTarGZip {
				cfg.SetNoUntarAfterDecompression(false)
			}

			err := ae.Unpacker(ctx, t, dst, src, cfg)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrFailedToUnpack, err)
			}
			return nil
		}

		return fmt.Errorf("%w: %q not in %q", ErrUnsupportedFileType, et, AvailableExtractors.Extensions())
	}

	header, reader, err := GetHeader(src)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToReadHeader, err)
	}

	var ext string
	if f, ok := src.(*os.File); ok {
		ext = filepath.Ext(f.Name())
	}

	unpacker := AvailableExtractors.GetUnpackFunction(header, ext)
	if unpacker != nil {
		err := unpacker(ctx, t, dst, reader, cfg)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToUnpack, err)
		}
		return nil
	}

	return ErrNoExtractorFound
}
