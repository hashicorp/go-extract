# go-extract
Secure extraction of any archive type.

## Ressources

* https://pypi.org/project/SecureZip/
* https://www.unforgettable.dk/

##  Feature collection

- [ ] extraction time exhaustion
- [ ] extraction size / zip bomb
- [ ] recursive extraction
- [ ] filetype detection based on magic bytes

## Intended filetypes

- [x] zip
    - [x] symlink inside archive
    - [x] symlink to outside is detected
    - [x] symlink with absolut path is detected
    - [x] file with path traversal is detected
    - [x] file with absolut path is detected
- [ ] slug
- [x] tar
    - [x] symlink inside archive
    - [x] symlink to outside is detected
    - [x] symlink with absolut path is detected
    - [x] file with path traversal is detected
    - [x] file with absolut path is detected
- [ ] gunzip
- [ ] tar.gz
- [ ] 7zip
- [ ] rar
- [ ] deb
- [ ] jar
- [ ] pkg
