# go-extract

[![Perform tests on unix and windows](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml) [![Security Scanner](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml) [![Heimdall](https://heimdall.hashicorp.services/api/v1/assets/go-extract/badge.svg?key=ad16a37b0882cb2e792c11a031b139227b23eabe137ddf2b19d10028bcdb79a8)](https://heimdall.hashicorp.services/site/assets/go-extract)

Secure file decompression and extraction of following types:

- Brotli
- Bzip2
- GZip
- LZ4
- Snappy
- Tar
- Xz
- Zip
- Zlib
- Zstandard

## Code Example

Add to `go.mod`:

```cli
GOPRIVATE=github.com/hashicorp/go-extract go get github.com/hashicorp/go-extract
```

Usage in code:

```go

import (
    ...
    "github.com/hashicorp/go-extract"
    "github.com/hashicorp/go-extract/config"
    "github.com/hashicorp/go-extract/telemetry"
    ...
)

...


    // open archive
    archive, _ := os.Open(...)

    // prepare context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), (time.Second * time.Duration(MaxExtractionTime)))
    defer cancel()

    // prepare logger
    logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
      Level: slog.LevelInfo,
    }))

    // setup telemetry hook
    telemetryToLog := func(ctx context.Context, td telemetry.Data) {
      logger.Info("extraction finished", "telemetryData", td)
    }

    // prepare config (these are the default values)
    config := config.NewConfig(
        config.WithCacheInMemory(false),              // cache to disk if input is a zip in a stream
        config.WithContinueOnError(false),            // fail on error
        config.WithContinueOnUnsupportedFiles(false), // don't on unsupported files
        config.WithCreateDestination(false),          // do not try to create specified destination
        config.WithDenySymlinkExtraction(false),      // allow symlink creation
        config.WithFollowSymlinks(false),             // do not follow symlinks during creation
        config.WithLogger(logger),                    // adjust logger (default: io.Discard)
        config.WithMaxExtractionSize(1 << (10 * 3)),  // limit to 1 Gb (disable check: -1)
        config.WithMaxFiles(1000),                    // only 1k files maximum (disable check: -1)
        config.WithMaxInputSize(1 << (10 * 3)),       // limit to 1 Gb (disable check: -1)
        config.WithNoUntarAfterDecompression(false),  // extract tar.gz combined
        config.WithOverwrite(false),                  // don't replace existing files
        config.WithPatterns("*.tf","modules/*.tf"),   // normally, no patterns predefined
        config.WithTelemetryHook(telemetryToLog),     // adjust hook to receive telemetry from extraction
    )

    // extract archive
    if err := extract.Unpack(ctx, archive, destinationPath, config); err != nil {
      // handle error
    }

...

```

> [!TIP]
> If the library is used in a cgroup memory limited execution environment to extract Zip archives that are cached in memory (`config.WithCacheInMemory(true)`), make sure that [`GOMEMLIMIT`](https://pkg.go.dev/runtime) is set in the execution environment to avoid `OOM` error.
>
> Example:
>
> ```shell
> $ export GOMEMLIMIT=1GiB
> ```

## CLI Tool

You can use this library on the command line with the `goextract` command.

### Installation

```cli
GOPRIVATE=github.com/hashicorp/go-extract go install github.com/hashicorp/go-extract/cmd/goextract@latest
```

### Manual Build and Installation

```cli
git clone git@github.com:hashicorp/go-extract.git
cd go-extract
make
make test
make install
```

### Usage

```cli
$ goextract -h
Usage: goextract <archive> [<destination>] [flags]

A secure extraction utility

Arguments:
  <archive>          Path to archive. ("-" for STDIN)
  [<destination>]    Output directory/file.

Flags:
  -h, --help                              Show context-sensitive help.
  -C, --continue-on-error                 Continue extraction on error.
  -S, --continue-on-unsupported-files     Skip extraction of unsupported files.
  -c, --create-destination                Create destination directory if it does not exist.
  -D, --deny-symlinks                     Deny symlink extraction.
  -F, --follow-symlinks                   [Dangerous!] Follow symlinks to directories during extraction.
      --max-files=1000                    Maximum files that are extracted before stop. (disable check: -1)
      --max-extraction-size=1073741824    Maximum extraction size that allowed is (in bytes). (disable check: -1)
      --max-extraction-time=60            Maximum time that an extraction should take (in seconds). (disable check: -1)
      --max-input-size=1073741824         Maximum input size that allowed is (in bytes). (disable check: -1)
  -N, --no-untar-after-decompression      Disable combined extraction of tar.gz.
  -O, --overwrite                         Overwrite if exist.
  -P, --pattern=PATTERN,...               Extracted objects need to match shell file name pattern.
  -T, --telemetry                         Print telemetry data to log after extraction.
  -v, --verbose                           Verbose logging.
  -V, --version                           Print release version information.
```

## Telemetry data

It is possible to collect telemetry data ether by specifying a telemetry hook via the config option `config.WithTelemetryHook(telemetryToLog)` or as a cli parameter `-T, --telemetry`.

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

## Feature collection

- Filetypes
  - [x] zip (/jar)
  - [x] tar
  - [x] gzip
  - [x] tar.gz
  - [x] brotli
  - [x] bzip2
  - [x] flate
  - [x] xz
  - [x] snappy
  - [ ] rar
  - [ ] 7zip
  - [x] zstandard
  - [x] zlib
  - [x] lz4
- [x] extraction size check
- [x] max num of extracted files
- [x] extraction time exhaustion
- [x] input file size limitations
- [x] context based cancelation
- [x] option pattern for configuration
- [x] `io.Reader` as source
- [x] symlink inside archive
- [x] symlink to outside is detected
- [x] symlink with absolute path is detected
- [x] file with path traversal is detected
- [x] file with absolute path is detected
- [x] filetype detection based on magic bytes
- [x] windows support
- [x] tests for gzip
- [x] function documentation
- [x] check for windows
- [x] Allow/deny symlinks in general
- [x] Telemetry call back function
- [x] Extraction filter with [unix file name patterns](https://pkg.go.dev/path/filepath#Match)
- [x] Cache input on disk (only relevant if `<archive>` is a zip archive, which read from a stream)
- [x] Cache alternatively optional input in memory (similar to caching on disk, only relevant for zip archives that are consumed from a stream)
- [ ] Handle passwords
- [ ] recursive extraction
- [ ] virtual fs as target

## References

- [SecureZip](https://pypi.org/project/SecureZip/)
- [42zip](https://www.unforgettable.dk/)
