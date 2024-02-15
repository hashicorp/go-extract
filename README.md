# go-extract

[![Perform tests on unix and windows](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml) [![Security Scanner](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml) [![Heimdall](https://heimdall.hashicorp.services/api/v1/assets/go-extract/badge.svg?key=ad16a37b0882cb2e792c11a031b139227b23eabe137ddf2b19d10028bcdb79a8)](https://heimdall.hashicorp.services/site/assets/go-extract)

Secure in-memory extraction of zip/tar/tar.gz/gz archive type.

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

    // setup metrics hook
    metricsToLog := func(ctx context.Context, metrics config.Metrics) {
      logger.Info("extraction finished", "metrics", metrics)
    }

    // prepare config (these are the default values)
    config := config.NewConfig(
        config.WithAllowSymlinks(true),               // allow symlink creation
        config.WithCacheInMemory(false),              // cache to disk if necessary
        config.WithContinueOnError(false),            // fail on error
        config.WithContinueOnUnsupportedFiles(false), // don't on unsupported files
        config.WithCreateDestination(false),          // do not try to create specified destination
        config.WithFollowSymlinks(false),             // do not follow symlinks during creation
        config.WithLogger(logger),                    // adjust logger (default: io.Discard)
        config.WithMaxExtractionSize(1 << (10 * 3)),  // limit to 1 Gb (disable check: -1)
        config.WithMaxFiles(1000),                    // only 1k files maximum (disable check: -1)
        config.WithMaxInputSize(1 << (10 * 3)),       // limit to 1 Gb (disable check: -1)
        config.WithMetricsHook(metricsToLog),         // adjust hook to receive metrics from extraction
        config.WithNoTarGzExtract(true),              // extract tar.gz combined
        config.WithOverwrite(false),                  // don't replace existing files
        config.WithPattern("*.tf","modules/*.tf"),    // no patterns predefined
    )

    // extract archive
    if err := extract.Unpack(ctx, archive, destinationPath, config); err != nil {
      // handle error
    }

...

```

> [!TIP]
> If the library is used in a cgroup memory limited execution environment to extract Zip archives that are cached in memory (`config.WithCacheInMemory(true)`), make sure that `[GOMEMLIMIT](https://pkg.go.dev/runtime#:~:text=GOMEMLIMIT%20is%20a%20numeric%20value%20in%20bytes%20with%20an%20optional%20unit%20suffix.%20The%20supported%20suffixes%20include%20B%2C%20KiB%2C%20MiB%2C%20GiB%2C%20and%20TiB)` is set in the execution environment to avoid `OOM` error.
> 
> Example:
> ```
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
Usage: goextract <archive> [<destination>]

A secure extraction utility

Arguments:
  <archive>          Path to archive. ("-" for STDIN)
  [<destination>]    Output directory/file.

Flags:
  -h, --help                              Show context-sensitive help.
  -I, --cache-in-memory                   Cache in memory instead of disc (only if necessary).
  -C, --continue-on-error                 Continue extraction on error.
  -S, --continue-on-unsupported-files     Skip extraction of unsupported files.
  -c, --create-destination                Create destination directory if it does not exist.
  -D, --deny-symlinks                     Deny symlink extraction.
  -F, --follow-symlinks                   [Dangerous!] Follow symlinks to directories during extraction.
      --max-files=1000                    Maximum files that are extracted before stop. (disable check: -1)
      --max-extraction-size=1073741824    Maximum extraction size that allowed is (in bytes). (disable check: -1)
      --max-extraction-time=60            Maximum time that an extraction should take (in seconds). (disable check: -1)
      --max-input-size=1073741824         Maximum input size that allowed is (in bytes). (disable check: -1)
  -M, --metrics                           Print metrics to log after extraction.
  -N, --no-tar-gz                         Disable combined extraction of tar.gz.
  -O, --overwrite                         Overwrite if exist.
  -P, --pattern=PATTERN,...               Extracted objects need to match shell file name pattern.
  -v, --verbose                           Verbose logging.
  -V, --version                           Print release version information.
```

## Feature collection

- Filetypes
  - [x] zip (/jar)
  - [x] tar
  - [x] gzip
  - [x] tar.gz
  - [ ] bzip2
  - [ ] 7zip
  - [ ] rar
  - [ ] deb
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
- [x] Metrics call back function
- [x] Extraction filter with file name patterns
- [x] Cache input on disk, if necessary
- [ ] Handle passwords
- [ ] recursive extraction
- [ ] virtual fs as target

## References

- [SecureZip](https://pypi.org/project/SecureZip/)
- [42zip](https://www.unforgettable.dk/)
