package extract

import (
	"archive/tar"
	"io"
	"log"
	"os"
)

// reference https://www.geeksforgeeks.org/time-sleep-function-in-golang-with-examples/

type Tar struct {
}

func (a *Tar) Extract(src, dst string) error {

	tarFile, err := os.Open(src)
	if err != nil {
		return err
	}

	tr := tar.NewReader(tarFile)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			// reached end of archive
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			// handle directory
			if err := createDir(dst, header.Name); err != nil {
				return err
			}
			continue

		// if it's a file create it
		case tar.TypeReg:

			if err := createFile(dst, header.Name, tr, os.FileMode(header.Mode)); err != nil {
				return err
			}

		// its a symlink !!
		case tar.TypeSymlink:
			// create link
			if err := createSymlink(dst, header.Name, header.Linkname); err != nil {
				return err
			}

		default:
			log.Printf("unspported filemode: %v", header.Typeflag)
		}
	}
}
