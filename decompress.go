// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package extract

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
)

type decompressionFunc func(io.Reader) (io.Reader, error)

func decompress(ctx context.Context, t Target, dst string, src io.Reader, cfg *Config, decFunc decompressionFunc, fileExt string) error {

	// prepare telemetry capturing
	// remark: do not defer TelemetryHook here, bc/ in case of tar.<compression>, the
	// tar extractor should submit the telemetry data
	cfg.Logger().Info("decompress", "fileExt", fileExt)
	m := &TelemetryData{ExtractedType: fileExt}
	defer cfg.TelemetryHook()(ctx, m)
	defer captureExtractionDuration(m, now())

	// limit input size
	limitedReader := newLimitErrorReader(src, cfg.MaxInputSize())
	defer captureInputSize(m, limitedReader)

	// start decompression
	decompressedStream, err := decFunc(limitedReader)
	if err != nil {
		return handleError(cfg, m, "cannot start decompression", err)
	}
	defer func() {
		if closer, ok := decompressedStream.(io.Closer); ok {
			closer.Close()
		}
	}()
	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(cfg, m, "context error", err)
	}

	// convert to peek header
	headerReader, err := newHeaderReader(decompressedStream, maxHeaderLength)
	if err != nil {
		return handleError(cfg, m, "cannot read uncompressed header", err)
	}

	// check if context is canceled
	if err := ctx.Err(); err != nil {
		return handleError(cfg, m, "context error", err)
	}

	// check if uncompressed stream is tar
	headerBytes := headerReader.PeekHeader()

	// check for tar header
	checkUntar := !cfg.NoUntarAfterDecompression()
	if checkUntar && isTar(headerBytes) {
		m.ExtractedType = fmt.Sprintf("tar.%s", fileExt) // combine types
		return processTar(ctx, t, headerReader, dst, cfg, m)
	}

	// determine name and decompress content
	inputName := ""
	if f, ok := src.(*os.File); ok {
		inputName = filepath.Base(f.Name())
	}
	dst, outputName := determineOutputName(t, dst, inputName, fmt.Sprintf(".%s", fileExt))
	cfg.Logger().Debug("determined output name", "name", outputName)
	n, err := createFile(t, dst, outputName, headerReader, cfg.CustomDecompressFileMode(), cfg.MaxExtractionSize(), cfg)
	m.ExtractionSize = n
	if err != nil {
		return handleError(cfg, m, "cannot create file", err)
	}
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
		{"maximum length 255", regexp.MustCompile(`^.{256,}$`)},
		{"limit to first 255 ascii characters", regexp.MustCompile(`[^\x00-\xFF]`)},
		{"exclude line break, feed and tab", regexp.MustCompile(`[\x0a\x0d\x09]`)},
	}

	if runtime.GOOS != "windows" {

		// regex with invalid unix filesystem characters, allowing unicode (128-255), excluding following character: / null byte backslash
		namingRestrictions = append(namingRestrictions,
			nameRestriction{"invalid character in filename (unix): null byte, slash, backslash", regexp.MustCompile(`[\x00/\\]`)},
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
	// defaultDecompressionName is the default name for the extracted content
	defaultDecompressionName = "goextract-decompressed-content"

	// defaultDecompressedSuffix is the suffix for the extracted content if
	// the filename does not end with a file extension
	defaultDecompressedSuffix = "decompressed"
)

// determineOutputName determines the output name and directory for the extracted content
func determineOutputName(t Target, dst string, inputName string, fileExt string) (string, string) {

	// check if dst is specified and not a directory
	if dst != "." && dst != "" {
		stat, err := t.Lstat(dst)

		// check if dst does not exist, then use it as directory and output name
		if os.IsNotExist(err) {
			return filepath.Dir(dst), filepath.Base(dst)
		}

		// check if stat is a symlink
		if stat != nil && stat.Mode()&os.ModeSymlink != 0 {
			stat, err = t.Stat(dst)
		}

		// check again if dst does not exist, then use it as directory and output name
		if os.IsNotExist(err) {
			return filepath.Dir(dst), filepath.Base(dst)
		}

		// check if dst is NOT a directory, then use it as directory
		// and output name (override might be necessary)
		if err == nil && stat != nil && !stat.IsDir() {
			return filepath.Dir(dst), filepath.Base(dst)
		}
	}

	// is src for decompression a file?
	if len(inputName) == 0 {
		return dst, defaultDecompressionName
	}

	// start with the input name
	newName := inputName

	// remove file extension
	if strings.HasSuffix(strings.ToLower(inputName), strings.ToLower(fileExt)) {
		newName = newName[:len(newName)-len(fileExt)]
	}

	// check if file extension has been removed, if not, add a suffix
	if newName == inputName {
		newName = fmt.Sprintf("%s.%s", inputName, defaultDecompressedSuffix)
	}

	// check newName is a valid utf8 string
	if !utf8.ValidString(newName) {
		return dst, defaultDecompressionName
	}

	// check if the new filename without the extension is valid and does not violate
	// any restrictions for the operating system
	// newNameBytes := []byte(newName)
	for _, restriction := range namingRestrictions {
		if restriction.Regex.FindStringIndex(newName) != nil {
			return dst, defaultDecompressionName
		}
	}

	// return the new name
	return dst, newName
}
