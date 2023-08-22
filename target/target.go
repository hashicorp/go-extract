package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

type Target interface {
	CreateSafeFile(config *config.Config, dstDir string, name string, reader io.Reader, mode fs.FileMode) error
	CreateSafeDir(config *config.Config, dstDir string, dirName string) error
	CreateSafeSymlink(config *config.Config, dstDir string, name string, linkTarget string) error
}
