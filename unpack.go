package extract

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/internal/extractor"
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

// Target is an interface that defines the methods that a target must implement
// so that the unpacking process can be done.
type Target extractor.Target

// NewMemoryTarget returns a new memory target that provides an in-memory filesystem.
func NewMemoryTarget() Target {
	return extractor.NewMemory()
}

// NewOSTarget returns a new OS target that uses the filesystem of the operating system.
func NewOSTarget() Target {
	return extractor.NewOS()
}

// Unpack unpacks the given source to the destination, according to the given configuration,
// using the default OS extractor. If an error occurs, it is returned.
func Unpack(ctx context.Context, src io.Reader, dst string, cfg *config.Config) error {
	return UnpackTo(ctx, extractor.NewOS(), dst, src, cfg)
}

// UnpackTo unpacks the given source to the destination, according to the given configuration,
// using the given extractor.Target. If an error occurs, it is returned.
func UnpackTo(ctx context.Context, t Target, dst string, src io.Reader, cfg *config.Config) error {
	if et := cfg.ExtractType(); len(et) > 0 {
		if ae, found := extractor.AvailableExtractors[et]; found {
			if et == extractor.FileExtensionTarGZip {
				cfg.SetNoUntarAfterDecompression(false)
			}

			err := ae.Unpacker(ctx, t, dst, src, cfg)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrFailedToUnpack, err)
			}
			return nil
		}

		return fmt.Errorf("%w: %q not in %q", ErrUnsupportedFileType, et, extractor.AvailableExtractors.Extensions())
	}

	header, reader, err := extractor.GetHeader(src)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToReadHeader, err)
	}

	var ext string
	if f, ok := src.(*os.File); ok {
		ext = filepath.Ext(f.Name())
	}

	unpacker := extractor.AvailableExtractors.GetUnpackFunction(header, ext)
	if unpacker != nil {
		err := unpacker(ctx, t, dst, reader, cfg)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToUnpack, err)
		}
		return nil
	}

	return ErrNoExtractorFound
}
