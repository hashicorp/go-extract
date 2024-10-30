# go-extract

[![Perform tests on unix and windows](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml)
[![GoDoc](https://godoc.org/github.com/hashicorp/go-extract?status.svg)](https://godoc.org/github.com/hashicorp/go-extract)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

This library provides secure decompression and extraction for formats like 7-Zip, Brotli, Bzip2, GZip, LZ4, Rar (excluding symlinks), Snappy, Tar, Xz, Zip, Zlib, and Zstandard. It safeguards against resource exhaustion, path traversal, and symlink attacks. Additionally, it offers various configuration options and collects telemetry data during extraction.

## Installation Instructions

Add [hashicorp/go-extract](https://github.com/hashicorp/go-extract) as a dependency to your project:

```cli
go get github.com/hashicorp/go-extract
```

Build [hashicorp/go-extract](https://github.com/hashicorp/go-extract) from source and install it to the system as a command-line utility:

```cli
git clone git@github.com:hashicorp/go-extract.git
cd go-extract
make
make test
make install
```

Install [hashicorp/go-extract](https://github.com/hashicorp/go-extract) directly from GitHub:

```cli
go install github.com/hashicorp/go-extract/cmd/goextract@latest
```

## Usage Examples

These examples demonstrate how to use [hashicorp/go-extract](https://github.com/hashicorp/go-extract) both as a library and as a command-line utility.

### Command-line Utility

The `goextract` command-line utility offers all available configuration options via dedicated flags.

```shell
$ goextract -h
Usage: goextract <archive> [<destination>] [flags]

A secure extraction utility

Arguments:
  <archive>          Path to archive. ("-" for STDIN)
  [<destination>]    Output directory/file.

Flags:
  -h, --help                               Show context-sensitive help.
  -C, --continue-on-error                  Continue extraction on error.
  -S, --continue-on-unsupported-files      Skip extraction of unsupported files.
  -c, --create-destination                 Create destination directory if it does not exist.
      --custom-create-dir-mode=750         File mode for created directories, which are not listed in the archive. (respecting umask)
      --custom-decompress-file-mode=640    File mode for decompressed files. (respecting umask)
  -D, --deny-symlinks                      Deny symlink extraction.
  -F, --follow-symlinks                    [Dangerous!] Follow symlinks to directories during extraction.
      --max-files=1000                     Maximum files (including folder and symlinks) that are extracted before stop. (disable check: -1)
      --max-extraction-size=1073741824     Maximum extraction size that allowed is (in bytes). (disable check: -1)
      --max-extraction-time=60             Maximum time that an extraction should take (in seconds). (disable check: -1)
      --max-input-size=1073741824          Maximum input size that allowed is (in bytes). (disable check: -1)
  -N, --no-untar-after-decompression       Disable combined extraction of tar.gz.
  -O, --overwrite                          Overwrite if exist.
  -P, --pattern=PATTERN,...                Extracted objects need to match shell file name pattern.
  -T, --telemetry                          Print telemetry data to log after extraction.
  -t, --type=""                            Type of archive. (7z, br, bz2, gz, lz4, rar, sz, tar, tgz, xz, zip, zst, zz)
  -v, --verbose                            Verbose logging.
  -V, --version                            Print release version information.
```

### Library

The simplest way to use the library is to call the `extract.Unpack` function with the default configuration. This function extracts the contents from an `io.Reader` to the specified destination on the local filesystem.

```go
// Unpack the archive
if err := extract.Unpack(ctx, archive, dst, config.NewConfig()); err != nil {
    // Handle error
    log.Fatalf("Failed to unpack archive: %v", err)
}
```

## Configuration

When calling the `extract.Unpack(..)` function, we need to provide `config` object that contains all available configuration.

```golang
  // process cli params
  cfg := config.NewConfig(
    config.WithContinueOnError(..),
    config.WithContinueOnUnsupportedFiles(..),
    config.WithCreateDestination(..),
    config.WithCustomCreateDirMode(..),
    config.WithCustomDecompressFileMode(..),
    config.WithDenySymlinkExtraction(..),
    config.WithExtractType(..),
    config.WithFollowSymlinks(..),
    config.WithLogger(..),
    config.WithMaxExtractionSize(..),
    config.WithMaxFiles(..),
    config.WithMaxInputSize(..),
    config.WithNoUntarAfterDecompression(..),
    config.WithOverwrite(..),
    config.WithPatterns(..),
    config.WithTelemetryHook(..),
  )

[..]

  if err := extract.Unpack(ctx, archive, dst, cfg); err != nil {
    log.Println(fmt.Errorf("error during extraction: %w", err))
    os.Exit(-1)
  }
```

## Telemetry

Telemetry data can be collected by specifying a telemetry hook in the configuration. This hook receives the collected telemetry data at the end of each extraction.

```golang
// create new config
cfg := NewConfig(
  WithTelemetryHook(func(ctx context.Context, m *telemetry.Data) {
    // handle telemetry data
  }),
)
```

Here is an example collected telemetry data for the extraction of [`terraform-aws-iam-5.34.0.tar.gz`](https://github.com/terraform-aws-modules/terraform-aws-iam/releases/tag/v5.34.0):

```json
{
  "LastExtractionError": "",
  "ExtractedDirs": 51,
  "ExtractionDuration": 48598584,
  "ExtractionErrors": 0,
  "ExtractedFiles": 241,
  "ExtractionSize": 539085,
  "ExtractedSymlinks": 0,
  "ExtractedType": "tar+gzip",
  "InputSize": 81477,
  "PatternMismatches": 0,
  "UnsupportedFiles": 0,
  "LastUnsupportedFile": ""
}
```

## Extraction targets

### Operating System (OS)

Interact with the local operating system to create files, directories, and symlinks. Extracted entries can be accessed later using the `os.*` API calls.

```golang
// prepare destination and config
o := extract.NewOSTarget()
dst := "output/"
cfg := config.NewConfig()

// unpack
if err := extract.UnpackTo(ctx, o, dst, archive, cfg); err != nil {
    // handle error
}

// Walk the local filesystem
localFs := os.DirFS(dst)
if err := fs.WalkDir(localFs, ".", func(path string, d fs.DirEntry, err error) error {
    // process path, d and err
    return nil
}); err != nil {
    // handle error
}
```

### Memory

Extract archives directly into memory, supporting files, directories, and symlinks. Note that file permissions are not validated. Access the extracted entries by converting the target to [io/fs.FS](https://pkg.go.dev/io/fs#FS).

```golang
// prepare destination and config
m := extract.NewMemoryTarget()
dst := "" // extract to root of memory filesystem
cfg := config.NewConfig()

// unpack
if err := extract.UnpackTo(ctx, m, dst, archive, cfg); err != nil {
    // handle error
}

// Walk the memory filesystem
memFs := m.(fs.FS)
if err := fs.WalkDir(memFs, ".", func(path string, d fs.DirEntry, err error) error {
    fmt.Println(path)
    return nil
}); err != nil {
    fmt.Printf("failed to walk memory filesystem: %s", err)
    return
}
```

## Errors

The extraction process eventually fails, depending on the provided archive and input stream. If the extraction fails, at exist a set of default errors that might be thrown by the `extract.Unpack()` function.

```golang
if err := extract.Unpack(ctx, archive, dst, cfg); err != nil {
  switch {
  case errors.Is(err, extract.ErrNoExtractorFound):
    // handle no extractor found
  case errors.Is(err, extract.ErrUnsupportedFileType):
    // handle unsupported file type
  case errors.Is(err, extract.ErrFailedToReadHeader):
    // handle failed to read header
  case errors.Is(err, extract.ErrFailedToUnpack):
    // handle failed to unpack
  default:
    // handle other error
  }
}
```
