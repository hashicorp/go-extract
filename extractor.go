package extract

import (
	"context"

	"github.com/hashicorp/go-extract/config"
	"github.com/hashicorp/go-extract/target"
)

type ExtractorOption func(*Extractor)

type Extractor interface {
	Unpack(ctx context.Context, src string, dst string) error
	FileSuffix() string
	SetConfig(config *config.Config)
	SetTarget(target *target.Target)

	// TODO(jan): detect filetype based on magic bytes
	// MagicBytes() [][]byte
}

func WithConfig(config *config.Config) ExtractorOption {
	return func(e *Extractor) {
		(*e).SetConfig(config)
	}
}

func WithTarget(target *target.Target) ExtractorOption {
	return func(e *Extractor) {
		(*e).SetTarget(target)
	}
}
