package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// reference https://www.geeksforgeeks.org/time-sleep-function-in-golang-with-examples/

type Tar struct {
	config     *config.Config
	fileSuffix string
}

func NewTar(config *config.Config) *Tar {
	return &Tar{fileSuffix: ".tar", config: config}
}

func (t *Tar) FileSuffix() string {
	return t.fileSuffix
}

func (t *Tar) Config() *config.Config {
	return t.config
}

func (t *Tar) Unpack(ctx context.Context, src string, dst string) error {

	target := &target.Os{}

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
		fileCounter++
		if err := t.config.CheckMaxFiles(fileCounter); err != nil {
			return err
		}

		// check the file type
		switch hdr.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			// handle directory
			if err := target.CreateSafeDir(t.config, dst, hdr.Name); err != nil {
				return err
			}
			continue

		// if it's a file create it
		case tar.TypeReg:

			if err := t.config.CheckFileSize(hdr.Size); err != nil {
				return err
			}

			if err := target.CreateSafeFile(t.config, dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {
				return err
			}

		// its a symlink !!
		case tar.TypeSymlink:
			// create link
			if err := target.CreateSafeSymlink(t.config, dst, hdr.Name, hdr.Linkname); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unspported filemode: %v", hdr.Typeflag)
		}

	}
}
