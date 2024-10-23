# go-extract

[![Perform tests on unix and windows](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/testing.yml)

Secure  decompression and extraction for 7-Zip, Brotli, Bzip2, GZip, LZ4, Rar (without symlinks), Snappy, Tar, Xz, Zip, Zlib and Zstandard.

Go-extract prevents against exhaustion, path traversal and symlink attacks. The extraction offers various configuration options and collects telemetry data.

## Code Example

Add to `go.mod`:

```cli
go get github.com/hashicorp/go-extract
```

Usage in code:

```go
// prepare context, config and destination
ctx := context.Background()
dst := "output/"
cfg := config.NewConfig()

// unpack
if err := extract.Unpack(ctx, archive, dst, cfg); err != nil {
    // handle error
}

```

> [!TIP]
> If the library is used in a cgroup memory limited execution environment to extract Zip archives that are cached in memory (`config.WithCacheInMemory(true)`), make sure that [`GOMEMLIMIT`](https://pkg.go.dev/runtime) is set in the execution environment to avoid `OOM` error.
>
> Example:
>
> ```shell
> export GOMEMLIMIT=1GiB
> ```

## CLI Tool

You can use this library on the command line with the `goextract` command.

### Installation

```cli
go install github.com/hashicorp/go-extract/cmd/goextract@latest
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

## Extraction targets

### Operating System (OS)

Interact with the local operating system to, to create files, directories and symlinks.
Extracted entries can be accessed afterwards by `os.*` API calls.

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

Extract archives to memory by using the `target.Memory` implementation. Files, directories and symlinks
are supported. File permissions are not validated. Extracted entries are accessed ether via the call of `m.Open(..)`
or via a map key. Symlink semantically not processed by the implementation.

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

## Telemetry data

It is possible to collect telemetry data ether by specifying a telemetry hook via the config option or as a cli parameter.

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

## References

- [SecureZip](https://pypi.org/project/SecureZip/)
- [42zip](https://www.unforgettable.dk/)
- [google/safearchive](https://github.com/google/safearchive)
