package extract

import (
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

// ExtractorOption is a function pointer type for implementation of the option pattern
type ExtractorOption func(Extractor)

// Extractor is an interface and defines all functions that needs to be implemented by an extraction engine.
type Extractor interface {
	// Unpack is the main entrypoint to an extraction engine that takes the contents from src and extracts them to dst.
	Unpack(ctx context.Context, src io.Reader, dst string) error

	// SetConfig sets the config for an extraction engine.
	SetConfig(config *config.Config)

	// SetTarget sets target as a target in an extraction engine.
	SetTarget(target target.Target)

	// MagicBytes returns magic bytes that are used to identify the filetype.
	MagicBytes() [][]byte

	// Offset returns the offset of the magic bytes from MagicBytes().
	Offset() int
}

// WithConfig sets config as config in an Extractor
func WithConfig(config *config.Config) ExtractorOption {
	return func(e Extractor) {
		e.SetConfig(config)
	}
}

// WithTarget sets target as target in an Extractor
func WithTarget(target target.Target) ExtractorOption {
	return func(e Extractor) {
		e.SetTarget(target)
	}
}
