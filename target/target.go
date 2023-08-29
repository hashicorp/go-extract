package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

type TargetOption func(Target)

type Target interface {
	CreateSafeFile(dstDir string, name string, reader io.Reader, mode fs.FileMode) error
	CreateSafeDir(dstDir string, dirName string) error
	CreateSafeSymlink(dstDir string, name string, linkTarget string) error
	SetConfig(config *config.Config)
}

func WithConfig(config *config.Config) TargetOption {
	return func(target Target) {
		target.SetConfig(config)
	}
}
