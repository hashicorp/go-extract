package extract

import (
	"archive/tar"
	"io"
	"log"
	"os"
)

// reference https://www.geeksforgeeks.org/time-sleep-function-in-golang-with-examples/

type Tar struct {
	fileSuffix string
}

func NewTar() *Tar {
	return &Tar{fileSuffix: ".tar"}
}

func (t *Tar) FileSuffix() string {
	return t.fileSuffix
}

func (t *Tar) Extract(e *Extract, src, dst string) error {

	tarFile, err := os.Open(src)
	if err != nil {
		return err
	}

	var fileCounter int64

	tr := tar.NewReader(tarFile)

	for {
		hdr, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			// reached end of archive
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case hdr == nil:
			continue
		}

		// check for to many files in archive
		if err := e.incrementAndCheckMaxFiles(&fileCounter); err != nil {
			return err
		}

		// check the file type
		switch hdr.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			// handle directory
			if err := e.createDir(dst, hdr.Name); err != nil {
				return err
			}
			continue

		// if it's a file create it
		case tar.TypeReg:

			if err := e.checkFileSize(hdr.Size); err != nil {
				return err
			}

			if err := e.createFile(dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {
				return err
			}

		// its a symlink !!
		case tar.TypeSymlink:
			// create link
			if err := e.createSymlink(dst, hdr.Name, hdr.Linkname); err != nil {
				return err
			}

		default:
			log.Printf("unspported filemode: %v", hdr.Typeflag)
		}

	}
}
