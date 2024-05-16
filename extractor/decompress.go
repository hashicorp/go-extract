package extractor

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/telemetry"
)

type decompressionFunction func(io.Reader, *config.Config) (io.Reader, error)

func decompress(ctx context.Context, src io.Reader, dst string, c *config.Config, decom decompressionFunction, fileExt string) error {

	// prepare telemetry capturing
	// remark: do not defer TelemetryHook here, bc/ in case of tar.<compression>, the
	// tar extractor should submit the telemetry data
	c.Logger().Info("decompress", "fileExt", fileExt)
	m := &telemetry.Data{ExtractedType: fileExt}
	defer c.TelemetryHook()(ctx, m)
	defer captureExtractionDuration(m, now())

	// limit input size
	limitedReader := NewLimitErrorReader(src, c.MaxInputSize())
	defer captureInputSize(m, limitedReader)

	// start decompression
	decompressedStream, err := decom(limitedReader, c)
	if err != nil {
		return handleError(c, m, "cannot start decompression", err)
	}
	defer func() {
		if closer, ok := decompressedStream.(io.Closer); ok {
			closer.Close()
		}
	}()
	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, m, "context error", err)
	}

	// convert to peek header
	headerReader, err := NewHeaderReader(decompressedStream, MaxHeaderLength)
	if err != nil {
		return handleError(c, m, "cannot read uncompressed header", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(c, m, "context error", err)
	}

	// check if uncompressed stream is tar
	headerBytes := headerReader.PeekHeader()

	// check for tar header
	checkUntar := !c.NoUntarAfterDecompression()
	if checkUntar && IsTar(headerBytes) {
		m.ExtractedType = fmt.Sprintf("tar.%s", fileExt) // combine types
		return unpackTar(ctx, headerReader, dst, c, m)
	}

	// determine name and decompress content
	dst, outputName := determineOutputName(dst, src, fmt.Sprintf(".%s", fileExt))
	c.Logger().Debug("determined output name", "name", outputName)
	if err := unpackTarget.CreateSafeFile(c, dst, outputName, headerReader, c.CustomDecompressFileMode()); err != nil {
		return handleError(c, m, "cannot create file", err)
	}

	// capture telemetry
	stat, err := os.Stat(filepath.Join(dst, outputName))
	if err != nil {
		return handleError(c, m, "cannot stat file", err)
	}
	m.ExtractionSize = stat.Size()
	m.ExtractedFiles++

	// finished
	return nil

}

// init initializes the	extractor package and prepares the filename restriction regex
func init() {

	namingRestrictions = []nameRestriction{
		{"empty name", regexp.MustCompile(`^$`)},
		{"current directory", regexp.MustCompile(`^\.$`)},
		{"parent directory", regexp.MustCompile(`^\.\.$`)},
		{"maximum 255 characters", regexp.MustCompile(`^.{256,}$`)},
	}

	if runtime.GOOS != "windows" {

		// regex with invalid unix filesystem characters, allowing unicode (128-255), excluding following character: /
		namingRestrictions = append(namingRestrictions,
			nameRestriction{"invalid characters (unix)", regexp.MustCompile(`^.*[\x00/].*$`)},
		)

	}

	// check for invalid characters
	if runtime.GOOS == "windows" {

		// regex with invalid windows filesystem characters, allowing unicode (128-255), excluding control characters, and the following characters: <>:"/\\|?*e
		// https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
		namingRestrictions = append(namingRestrictions, nameRestriction{
			"invalid characters (windows)", regexp.MustCompile(`^.*[\x00-\x1f<>:"/\\|?*].*$`),
		})

		// known reserved names on windows, "(?i)" is case-insensitive
		namingRestrictions = append(namingRestrictions,
			nameRestriction{"reserved name", regexp.MustCompile(`^(?i)CON$`)},
			nameRestriction{"reserved name", regexp.MustCompile(`^(?i)PRN$`)},
			nameRestriction{"reserved name", regexp.MustCompile(`^(?i)AUX$`)},
			nameRestriction{"reserved name", regexp.MustCompile(`^(?i)NUL$`)},
			nameRestriction{"reserved name", regexp.MustCompile(`^(?i)COM[0-9]+$`)},
			nameRestriction{"reserved name", regexp.MustCompile(`^(?i)LPT[0-9]+$`)},
			nameRestriction{"reserved name", regexp.MustCompile(`^(\s|\.)+$`)})
	}

}

// nameRestriction is a struct that contains the name of the restriction and the regex to check for it
type nameRestriction struct {
	RestrictionName string
	Regex           *regexp.Regexp
}

// namingRestrictions is a list of restrictions for filenames, depending on the operating system
var namingRestrictions []nameRestriction

const (
	// DEFAULT_NAME is the default name for the extracted content
	DEFAULT_NAME = "goextract-decompressed-content"

	// SUFFIX is the suffix for the extracted content if
	// the filename does not end with a file extension
	SUFFIX = "decompressed"
)

// determineOutputName determines the output name and directory for the extracted content
func determineOutputName(dst string, src io.Reader, fileExt string) (string, string) {

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
		newName := name

		// check if the filename is ending with the file extension
		if strings.HasSuffix(name, fileExt) {
			newName = strings.TrimSuffix(name, fileExt)
		} else {
			newName = fmt.Sprintf("%s.%s", name, SUFFIX)
		}

		// check if the new filename without the extension is valid and does not violate
		// any restrictions for the operating system
		for _, restriction := range namingRestrictions {
			if restriction.Regex.MatchString(newName) {
				return dst, DEFAULT_NAME
			}
		}

		// if the filename is not ending with the suffix, use the suffix as output name
		return dst, newName
	}

	return dst, DEFAULT_NAME
}
