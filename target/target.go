package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract"
)

type Target interface {

	// CreateSafeFile creates name in dstDir with conte nt from reader and file
	// headers as provided in mode
	CreateSafeFile(config *extract.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode) error

	CreateSafeDir(dstDir string, dirName string) error

	CreateSymlink(dstDir string, name string, linkTarget string) error
}
