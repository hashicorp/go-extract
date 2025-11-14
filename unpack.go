// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

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

	// ErrUnsupportedFile is an error that indicates that the file is not supported.
	ErrUnsupportedFile = fmt.Errorf("extract: unsupported file")

	// ErrMaxFilesExceeded indicates that the maximum number of files is exceeded.
	ErrMaxFilesExceeded = fmt.Errorf("extract: maximum files exceeded")

	// ErrMaxExtractionSizeExceeded indicates that the maximum size is exceeded.
	ErrMaxExtractionSizeExceeded = fmt.Errorf("extract: maximum extraction size exceeded")
)

// Unpack unpacks the given source to the destination, according to the given configuration,
// using the default OS  If cfg is nil, the default configuration
// is used for extraction. If an error occurs, it is returned.
func Unpack(ctx context.Context, dst string, src io.Reader, cfg *Config) error {
	return UnpackTo(ctx, NewTargetDisk(), dst, src, cfg)
}

// UnpackTo unpacks the given source to the destination, according to the given configuration,
// using the given [Target]. If cfg is nil, the default configuration is used for extraction.
// If an error occurs, it is returned.
func UnpackTo(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config) error {
	if cfg == nil {
		cfg = NewConfig()
	}
	if et := cfg.ExtractType(); len(et) > 0 {
		if ae, found := availableExtractors[et]; found {
			if et == fileExtensionTarGZip {
				cfg.SetNoUntarAfterDecompression(false)
			}

			err := ae.Unpacker(ctx, t, dst, src, cfg)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrFailedToUnpack, err)
			}
			return nil
		}

		return fmt.Errorf("%w: %q not in %q", ErrUnsupportedFileType, et, availableExtractors.Extensions())
	}

	header, reader, err := getHeader(src)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToReadHeader, err)
	}

	var name string
	if f, ok := src.(*os.File); ok {
		name = filepath.Ext(f.Name())
	}

	unpacker := availableExtractors.GetUnpackFunction(header, name)
	if unpacker != nil {
		err := unpacker(ctx, t, dst, reader, cfg)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToUnpack, err)
		}
		return nil
	}

	return ErrNoExtractorFound
}

// HasKnownArchiveExtension returns true if the given name has a known archive extension.
func HasKnownArchiveExtension(name string) bool {
	return availableExtractors.GetUnpackFunctionByFileName(name) != nil
}
