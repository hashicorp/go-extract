package extract

import (
	"context"
	"io"

	"github.com/hashicorp/go-extract/config"
)

// Extractor is an interface and defines all functions that needs to be implemented by an extraction engine.
type Extractor interface {
	// Unpack is the main entrypoint to an extraction engine that takes the contents from src and extracts them to dst.
	Unpack(ctx context.Context, src io.Reader, dst string, config *config.Config) error
}
