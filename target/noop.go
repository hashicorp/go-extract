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
func (n *Noop) CreateSafeDir(dst string, name string, mode fs.FileMode, cfg *config.Config) error {
	return nil
}

// CreateSafeFile does nothing.
func (n *Noop) CreateSafeFile(dst string, name string, src io.Reader, mode fs.FileMode, cfg *config.Config) error {
	return nil
}

// CreateSafeSymlink does nothing.
func (n *Noop) CreateSafeSymlink(dst string, name string, target string, cfg *config.Config) error {
	return nil
}
