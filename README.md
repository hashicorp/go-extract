# go-extract
Secure extraction of any archive type.

## Ressources

* https://pypi.org/project/SecureZip/
* https://www.unforgettable.dk/

##  Feature collection

- [x] extraction size check
- [x] max num of extracted files
- [x] extraction time exhaustion
- [ ] go tests

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
