package extract

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/extractor"
)

// Unpack reads data from src, identifies if its a known archive type. If so, dst is unpacked
// in dst. opts can be given to adjust the config.
func Unpack(ctx context.Context, src io.Reader, dst string, c *config.Config) error {

	// check if type is set
	if len(c.ExtractType()) > 0 {
		if ae, found := extractor.AvailableExtractors[c.ExtractType()]; found {
			if c.ExtractType() == extractor.FileExtensionTarGZip {
				c.SetNoUntarAfterDecompression(false)
			}
			return ae.Unpacker(ctx, src, dst, c)
		}

		//
		return fmt.Errorf("not supported file extension %s", c.ExtractType())
	}

	// read headerReader to identify archive type
	header, reader, err := getHeader(src)
	if err != nil {
		return fmt.Errorf("failed to read header: %s", err)
	}

	// find extractor by header
	if unpacker := GetUnpackFunction(header); unpacker != nil {
		return unpacker(ctx, reader, dst, c)
	}

	// find extractor by file extension
	if fin, ok := src.(*os.File); ok {
		if unpacker := GetUnpackFunctionByFileName(fin.Name()); unpacker != nil {
			return unpacker(ctx, reader, dst, c)
		}
	}

	// perform extraction with identified reader
	return fmt.Errorf("no supported archive type ether not detected")
}

// getHeader reads the header from src and returns it. If src is a io.Seeker, the header is read
// directly from the reader and the reader gets reset. If src is not a io.Seeker, the header is read
// and transformed into a HeaderReader, which is returned as the second return value. If an error
// occurs, the header is nil and the error is returned as the third return value
func getHeader(src io.Reader) ([]byte, io.Reader, error) {

	// check if source offers seek and preserve type of source
	if s, ok := src.(io.Seeker); ok {

		// allocate buffer for header
		header := make([]byte, extractor.MaxHeaderLength)

		// read header from source
		_, err := src.Read(header)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read header: %s", err)
		}
		// reset reader
		_, err = s.Seek(0, io.SeekStart)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to reset reader: %s", err)
		}
		return header, src, nil
	}

	headerReader, err := extractor.NewHeaderReader(src, extractor.MaxHeaderLength)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create header reader: %s", err)
	}
	return headerReader.PeekHeader(), headerReader, nil
}

// GetUnpackFunction identifies the correct extractor based on magic bytes.
func GetUnpackFunction(data []byte) extractor.UnpackFunc {
	// find extractor with longest suffix match
	for _, ex := range extractor.AvailableExtractors {
		if ex.HeaderCheck(data) {
			return ex.Unpacker
		}
	}

	// no matching reader found
	return nil
}

// GetUnpackFunctionByFileName identifies the correct extractor based on file extension.
func GetUnpackFunctionByFileName(src string) extractor.UnpackFunc {
	// get file extension from file name
	src = strings.ToLower(src)
	src = filepath.Ext(src)
	src = strings.Replace(src, ".", "", -1) // remove leading dot if the file extension is the only part of the file name (e.g. ".tar")

	if ae, found := extractor.AvailableExtractors[src]; found {
		return ae.Unpacker
	}

	// no matching reader found
	return nil
}

// IsKnownArchiveFileExtension checks if the given file extension is a known archive file extension.
func IsKnownArchiveFileExtension(src string) bool {
	return GetUnpackFunctionByFileName(src) != nil
}

// Available file types
const (
	FileType7zip    = extractor.FileExtension7zip
	FileTypeBrotli  = extractor.FileExtensionBrotli
	FileTypeBzip2   = extractor.FileExtensionBzip2
	FileTypeGZip    = extractor.FileExtensionGZip
	FileTypeLZ4     = extractor.FileExtensionLZ4
	FileTypeSnappy  = extractor.FileExtensionSnappy
	FileTypeTar     = extractor.FileExtensionTar
	FileTypeTarGZip = extractor.FileExtensionTarGZip
	FileTypeXz      = extractor.FileExtensionXz
	FileTypeZIP     = extractor.FileExtensionZIP
	FileTypeZlib    = extractor.FileExtensionZlib
	FileTypeZstd    = extractor.FileExtensionZstd
)

// ValidTypes returns a string with all available types.
func ValidTypes() string {
	var types []string
	for t := range extractor.AvailableExtractors {
		types = append(types, t)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}
