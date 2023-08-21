package extract

import (
	"context"

	"github.com/hashicorp/go-extract/config"
)

type ExtractorOption func(*Extractor)

type Extractor interface {
	Unpack(ctx context.Context, src string, dst string) error
	FileSuffix() string
	Config() *config.Config
}
