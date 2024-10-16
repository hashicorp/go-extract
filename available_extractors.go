package extract

import (
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
	"github.com/hashicorp/go-extract/target"
)

// init calculates the maximum header length
func init() {
	for _, ex := range availableExtractors {
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

// unpackFunc is a function that extracts the contents from src and extracts them to dst.
type unpackFunc func(context.Context, target.Target, string, io.Reader, *config.Config) error

// headerCheck is a function that checks if the given header matches the expected magic bytes.
type headerCheck func([]byte) bool

type availableExtractor struct {
	Unpacker    unpackFunc
	HeaderCheck headerCheck
	MagicBytes  [][]byte
	Offset      int
}

// availableExtractors is collection of new extractor functions with
// the required magic bytes and potential offset
var availableExtractors = map[string]availableExtractor{
	extractor.FileExtension7zip: {
		Unpacker:    extractor.Unpack7Zip,
		HeaderCheck: extractor.Is7zip,
		MagicBytes:  extractor.MagicBytes7zip,
	},
	extractor.FileExtensionBrotli: {
		Unpacker:    extractor.UnpackBrotli,
		HeaderCheck: extractor.IsBrotli,
	},
	extractor.FileExtensionBzip2: {
		Unpacker:    extractor.UnpackBzip2,
		HeaderCheck: extractor.IsBzip2,
		MagicBytes:  extractor.MagicBytesBzip2,
	},
	extractor.FileExtensionGZip: {
		Unpacker:    extractor.UnpackGZip,
		HeaderCheck: extractor.IsGZip,
		MagicBytes:  extractor.MagicBytesGZip,
	},
	extractor.FileExtensionLZ4: {
		Unpacker:    extractor.UnpackLZ4,
		HeaderCheck: extractor.IsLZ4,
		MagicBytes:  extractor.MagicBytesLZ4,
	},
	extractor.FileExtensionSnappy: {
		Unpacker:    extractor.UnpackSnappy,
		HeaderCheck: extractor.IsSnappy,
		MagicBytes:  extractor.MagicBytesSnappy,
	},
	extractor.FileExtensionTar: {
		Unpacker:    extractor.UnpackTar,
		HeaderCheck: extractor.IsTar,
		MagicBytes:  extractor.MagicBytesTar,
		Offset:      extractor.OffsetTar,
	},
	extractor.FileExtensionTarGZip: {
		Unpacker:    extractor.UnpackGZip,
		HeaderCheck: extractor.IsGZip,
		MagicBytes:  extractor.MagicBytesGZip,
	},
	extractor.FileExtensionXz: {
		Unpacker:    extractor.UnpackXz,
		HeaderCheck: extractor.IsXz,
		MagicBytes:  extractor.MagicBytesXz,
	},
	extractor.FileExtensionZIP: {
		Unpacker:    extractor.UnpackZip,
		HeaderCheck: extractor.IsZip,
		MagicBytes:  extractor.MagicBytesZIP,
	},
	extractor.FileExtensionZlib: {
		Unpacker:    extractor.UnpackZlib,
		HeaderCheck: extractor.IsZlib,
		MagicBytes:  extractor.MagicBytesZlib,
	},
	extractor.FileExtensionZstd: {
		Unpacker:    extractor.UnpackZstd,
		HeaderCheck: extractor.IsZstd,
		MagicBytes:  extractor.MagicBytesZstd,
	},
	extractor.FileExtensionRar: {
		Unpacker:    extractor.UnpackRar,
		HeaderCheck: extractor.IsRar,
		MagicBytes:  extractor.MagicBytesRar,
	},
}

// maxHeaderLength is the maximum header length of all extractors
var maxHeaderLength int
