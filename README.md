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

    // prepare config
    config := config.NewConfig(
        config.WithContinueOnError(false),            // fail on error
        config.WithDenySymlinks(false),               // allow symlink creation
        config.WithFollowSymlinks(false),             // do not follow symlinks during creation
        config.WithOverwrite(false),                  // dont replace existing files
        config.WithMaxExtractionTime(60),             // stop after 1 minute
        config.WithMaxExtractionSize(1 << (10 * 3)),  // limit to 1 Gb
        config.WithMaxFiles(1000),                    // only 1k files maximum
        config.WithVerbose(false),                    // dont show details
    )

    extractOptions := []extract.ExtractorOption{
        extract.WithConfig(config),
    }

    // extract archive
    if err := extract.Unpack(ctx, archive, destination, extractOptions...); err != nil {
        // handle error
    }

...

```

## Cli Tool

The libraray can also be used directly on the cli `extract`.

### Installation

```cli
GOPRIVATE=github.com/hashicorp/go-extract go install github.com/hashicorp/go-extract@latest
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

Arguments:
  <archive>          Path to archive.
  [<destination>]    Output directory/file

Flags:
  -h, --help                              Show context-sensitive help.
  -f, --force                             Force extraction and overwrite if exist
      --max-files=1000                    Maximum files that are extracted before stop
      --max-extraction-size=1073741824    Maximum extraction size that allowed is (in bytes)
      --max-extraction-time=60            Maximum time that an extraction should take (in seconds)
  -v, --verbose                           Verbose logging.
  -V, --version                           Print release version information.
```

## Feature collection

- Filetypes
  - [x] zip (/jar)
  - [x] tar
  - [x] gunzip
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
- [x] symlink with absolut path is detected
- [x] file with path traversal is detected
- [x] file with absolut path is detected
- [x] filetype detection based on magic bytes
- [x] windows support
- [x] tests for gunzip
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
