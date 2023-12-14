# go-extract

[![test linux](https://github.com/hashicorp/go-extract/actions/workflows/test-linux.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/test-linux.yml) [![test windows](https://github.com/hashicorp/go-extract/actions/workflows/test-windows.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/test-windows.yml) [![Security Scanner](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml) [![Heimdall](https://heimdall.hashicorp.services/api/v1/assets/go-extract/badge.svg?key=ad16a37b0882cb2e792c11a031b139227b23eabe137ddf2b19d10028bcdb79a8)](https://heimdall.hashicorp.services/site/assets/go-extract)

Secure extraction of zip/tar/tar.gz/gz archive type.

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

    ctx := context.Background()

    // open archive
    archive, _ := os.Open(...)

    // prepare config (these are the default values)
    config := config.NewConfig(
        config.WithAllowSymlinks(true),               // allow symlink creation
        config.WithContinueOnError(false),            // fail on error
        config.WithFollowSymlinks(false),             // do not follow symlinks during creation
        config.WithMaxExtractionSize(1 << (10 * 3)),  // limit to 1 Gb (disable check: -1)
        config.WithMaxFiles(1000),                    // only 1k files maximum (disable check: -1)
        config.WithOverwrite(false),                  // don't replace existing files
        config.WithLogLevel(slog.Info),               // don't show log (log with setting to slog.Debug)
        config.WithLogger(slog.Default(),             // adjust logger
    )

    // prepare context with timeout
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(context.Background(), (time.Second * time.Duration(MaxExtractionTime)))
    defer cancel()

    // extract archive
    if err := extract.Unpack(ctx, archive, destinationPath, target.NewOs(), config); err != nil {
      // handle error
    }

...

```

## Cli Tool

The library can also be used directly on the cli `extract`.

### Installation

```cli
GOPRIVATE=github.com/hashicorp/go-extract go install github.com/hashicorp/go-extract/cmd/extract@latest
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
extract -h
Usage: extract <archive> [<destination>]

A secure extraction utility

Arguments:
  <archive>          Path to archive. ("-" for STDIN)
  [<destination>]    Output directory/file.

Flags:
  -h, --help                              Show context-sensitive help.
  -C, --continue-on-error                 Continue extraction on error.
  -D, --deny-symlinks                     Deny symlink extraction.
  -F, --follow-symlinks                   [Dangerous!] Follow symlinks to directories during extraction.
      --max-files=1000                    Maximum files that are extracted before stop. (disable check: -1)
      --max-extraction-size=1073741824    Maximum extraction size that allowed is (in bytes). (disable check: -1)
      --max-extraction-time=60            Maximum time that an extraction should take (in seconds). (disable check: -1)
  -O, --overwrite                         Overwrite if exist.
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
- [x] context based cancleation
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
- [ ] PAX header extraction
- [ ] Allow/deny symlinks in general
- [ ] Allow/deny external directories!?
- [ ] Handle passwords
- [ ] recursive extraction
- [ ] virtual fs as target

## References

- [SecureZip](https://pypi.org/project/SecureZip/)
- [42zip](https://www.unforgettable.dk/)
