# go-extract

[![test linux](https://github.com/hashicorp/go-extract/actions/workflows/test-linux.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/test-linux.yml) [![test windows](https://github.com/hashicorp/go-extract/actions/workflows/test-windows.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/test-windows.yml) [![Security Scanner](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/secscan.yml) [![Heimdall](https://heimdall.hashicorp.services/api/v1/assets/go-extract/badge.svg?key=ad16a37b0882cb2e792c11a031b139227b23eabe137ddf2b19d10028bcdb79a8)](https://heimdall.hashicorp.services/site/assets/go-extract)

Secure extraction of any archive type.

## Ressources

* https://pypi.org/project/SecureZip/
* https://www.unforgettable.dk/

##  Feature collection

- [x] extraction size check
- [x] max num of extracted files
- [x] extraction time exhaustion
- [x] context based cancleation
- [x] option pattern for configuration
- [x] options pattern for target
- [x] byte stream as source
- [x] symlink inside archive
- [x] symlink to outside is detected
- [x] symlink with absolut path is detected
- [x] file with path traversal is detected
- [x] file with absolut path is detected
- [x] filetype detection based on magic bytes
- [x] tests for gunzip
- [x] function documentation
- [x] check for windows
- [ ] verify tests transfered from go-slug
    - [x] dot-dot as file name
    - [x] empty dir name
    - [ ] FIFO in tar
- [ ] Allow/deny symlinks in general
- [ ] Allow/deny external directories!?




## Intended filetypes

- [x] zip (/jar)
- [x] tar
- [x] gunzip
- [x] tar.gz

## Future extensions

- [ ] bzip2
- [ ] 7zip
- [ ] rar
- [ ] deb
- [ ] recursive extraction
- [ ] virtual fs as target
