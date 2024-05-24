package target

import (
	"io"
	"io/fs"

	"github.com/hashicorp/go-extract/config"
)

// Noop is a target that does nothing.
type Noop struct {
}

// NewNoopTarget returns a new Noop target.
func NewNoopTarget() *Noop {
	return &Noop{}
}

// CreateSafeDir does nothing.
func (n *Noop) CreateSafeDir(dstBase string, newDir string, mode fs.FileMode, config *config.Config) error {
	return nil
}

// CreateSafeFile does nothing.
func (n *Noop) CreateSafeFile(dstDir string, name string, reader io.Reader, mode fs.FileMode, config *config.Config) error {
	return nil
}

// CreateSafeSymlink does nothing.
func (n *Noop) CreateSafeSymlink(dstDir string, name string, linkTarget string, config *config.Config) error {
	return nil
}
