# go-extract

[![Go test](https://github.com/hashicorp/go-extract/actions/workflows/test.yml/badge.svg)](https://github.com/hashicorp/go-extract/actions/workflows/test.yml) [![Heimdall](https://heimdall.hashicorp.services/api/v1/assets/go-extract/badge.svg?key=ad16a37b0882cb2e792c11a031b139227b23eabe137ddf2b19d10028bcdb79a8)](https://heimdall.hashicorp.services/site/assets/go-extract)

Secure extraction of any archive type.

## Ressources

* https://pypi.org/project/SecureZip/
* https://www.unforgettable.dk/

##  Feature collection

- [x] extraction size check
- [x] max num of extracted files
- [x] extraction time exhaustion
- [x] go tests
- [x] option pattern for configuration
- [x] options pattern for target
- [ ] s3 as target <<-- skiped due too dependency reduction
- [ ] virtual fs as target
- [ ] byte stream as source

## Intended filetypes

- [x] zip
    - [x] symlink inside archive
    - [x] symlink to outside is detected
    - [x] symlink with absolut path is detected
    - [x] file with path traversal is detected
    - [x] file with absolut path is detected
- [x] tar
    - [x] symlink inside archive
    - [x] symlink to outside is detected
    - [x] symlink with absolut path is detected
    - [x] file with path traversal is detected
    - [x] file with absolut path is detected
- [ ] gunzip
- [ ] tar.gz

## Future extensions

- [ ] slug
- [ ] bzip2
- [ ] 7zip
- [ ] rar
- [ ] deb
- [ ] jar
- [ ] pkg

## Future features

- [ ] recursive extraction
- [ ] filetype detection based on magic bytes
