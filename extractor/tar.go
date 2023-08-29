package extractor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

type Tar struct {
	config     *config.Config
	fileSuffix string
	target     target.Target
	magicBytes [][]byte
	offset     int
}

func NewTar(config *config.Config) *Tar {
	// defaults
	const (
		fileSuffix = ".tar"
	)
	magicBytes := [][]byte{
		{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30},
		{0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x20, 0x00},
	}
	offset := 257

	target := target.NewOs()

	// instantiate
	tar := Tar{
		fileSuffix: fileSuffix,
		config:     config,
		target:     &target,
		magicBytes: magicBytes,
		offset:     offset,
	}

	// return the modified house instance
	return &tar
}

func (t *Tar) FileSuffix() string {
	return t.fileSuffix
}

func (t *Tar) SetConfig(config *config.Config) {
	t.config = config
}

func (t *Tar) SetTarget(target *target.Target) {
	t.target = *target
}

func (t *Tar) Offset() int {
	return t.offset
}

func (t *Tar) MagicBytes() [][]byte {
	return t.magicBytes
}

func (t *Tar) Unpack(ctx context.Context, src io.Reader, dst string) error {

	// start extraction without timer
	if t.config.MaxExtractionTime == -1 {
		return t.unpack(ctx, src, dst)
	}

	// prepare timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(t.config.MaxExtractionTime)*time.Second)
	defer cancel()

	exChan := make(chan error, 1)
	go func() {
		// extract files in tmpDir
		if err := t.unpack(ctx, src, dst); err != nil {
			exChan <- err
		}
		exChan <- nil
	}()

	// start extraction in on thread
	select {
	case err := <-exChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("maximum extraction time exceeded")
	}

	return nil

}

func (t *Tar) unpack(ctx context.Context, src io.Reader, dst string) error {

	var fileCounter int64

	tr := tar.NewReader(src)

	var extractionSize uint64

	for {

		// check if context is cancled
		if ctx.Err() != nil {
			return nil
		}

		// get next file
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

		// check if size is exceeded
		if err := t.config.CheckMaxFiles(fileCounter); err != nil {
			return err
		}

		// check the file type
		switch hdr.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			// handle directory
			if err := t.target.CreateSafeDir(t.config, dst, hdr.Name); err != nil {
				return err
			}
			continue

		// if it's a file create it
		case tar.TypeReg:

			// check extraction size
			extractionSize = extractionSize + uint64(hdr.Size)
			if err := t.config.CheckExtractionSize(int64(extractionSize)); err != nil {
				return err
			}

			if err := t.config.CheckExtractionSize(hdr.Size); err != nil {
				return err
			}

			if err := t.target.CreateSafeFile(t.config, dst, hdr.Name, tr, os.FileMode(hdr.Mode)); err != nil {
				return err
			}

		// its a symlink !!
		case tar.TypeSymlink:
			// create link
			if err := t.target.CreateSafeSymlink(t.config, dst, hdr.Name, hdr.Linkname); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unspported filemode: %v", hdr.Typeflag)
		}

	}

}

func (t *Tar) MagicBytesMatch(data []byte) bool {

	// check all possible magic bytes for extract engine
	for _, magicBytes := range t.magicBytes {

		// skip if data is smaler als tar header
		if t.offset+len(magicBytes) > len(data) {
			continue
		}

		// compare magic bytes with readed bytes
		var missMatch bool
		for idx, fileByte := range data[t.offset : t.offset+len(magicBytes)] {
			if fileByte != magicBytes[idx] {
				missMatch = true
				break
			}
		}

		// if no missmatch, successfull identified engine!
		if !missMatch {
			return true
		}

	}

	return false
}
