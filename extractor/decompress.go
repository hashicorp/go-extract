package extractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"

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
	inputName := ""
	if f, ok := src.(*os.File); ok {
		inputName = filepath.Base(f.Name())
	}
	dst, outputName := determineOutputName(dst, inputName, fmt.Sprintf(".%s", fileExt))
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
		{"limit to first 255 characters", regexp.MustCompile(`[^\x00-\xFF]`)},
		{"exclude line break, feed and tab", regexp.MustCompile(`[\x0a\x0d\x09]`)},
	}

	if runtime.GOOS != "windows" {

		// regex with invalid unix filesystem characters, allowing unicode (128-255), excluding following character: /
		namingRestrictions = append(namingRestrictions,
			nameRestriction{"invalid filename (unix): null byte", regexp.MustCompile(`[\x00]`)},
			nameRestriction{"invalid filename (unix): dangerous ascii characters", regexp.MustCompile(`[:/\<>|!?*'"&^$]`)},
			nameRestriction{"invalid filename (unix): backticks", regexp.MustCompile("`")},
		)

	}

	// check for invalid characters
	if runtime.GOOS == "windows" {

		// regex with invalid windows filesystem characters, allowing unicode (128-255), excluding control characters, and the following characters: <>:"/\\|?*e
		// https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
		namingRestrictions = append(namingRestrictions, nameRestriction{
			"invalid characters (windows)", regexp.MustCompile(`[\x00-\x1f<>:"/\\|?*]`),
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
	// DEFAULT_DECOMPRESSION_NAME is the default name for the extracted content
	DEFAULT_DECOMPRESSION_NAME = "goextract-decompressed-content"

	// DECOMPRESSED_SUFFIX is the suffix for the extracted content if
	// the filename does not end with a file extension
	DECOMPRESSED_SUFFIX = "decompressed"
)

// determineOutputName determines the output name and directory for the extracted content
func determineOutputName(dst string, inputName string, fileExt string) (string, string) {

	// check if dst is specified and not a directory
	if dst != "." && dst != "" {
		stat, err := os.Stat(dst)

		// check if dst does not exist, then use it as directory and output name
		if os.IsNotExist(err) {
			return filepath.Dir(dst), filepath.Base(dst)
		}

		// check if dst is NOT a directory, then use it as directory
		// and output name (override might be necessary)
		if err == nil && !stat.IsDir() {
			return filepath.Dir(dst), filepath.Base(dst)
		}
	}

	// is src for decompression a file?
	if len(inputName) > 0 {

		// start with the input name
		newName := inputName

		// remove file extension
		if strings.HasSuffix(strings.ToLower(inputName), strings.ToLower(fileExt)) {
			newName = newName[:len(newName)-len(fileExt)]
		}

		// check if file extension has been removed, if not, add a suffix
		if newName == inputName {
			newName = fmt.Sprintf("%s.%s", inputName, DECOMPRESSED_SUFFIX)
		}

		// check newName is a valid utf8 string
		if !utf8.ValidString(newName) {
			return dst, DEFAULT_DECOMPRESSION_NAME
		}

		// check if the new filename without the extension is valid and does not violate
		// any restrictions for the operating system
		// newNameBytes := []byte(newName)
		for _, restriction := range namingRestrictions {
			if restriction.Regex.FindStringIndex(newName) != nil {
				return dst, DEFAULT_DECOMPRESSION_NAME
			}
		}

		// return the new name
		return dst, newName
	}

	// return default name and provided directory
	return dst, DEFAULT_DECOMPRESSION_NAME
}
