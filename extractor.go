package extract

import "context"

type Extractor interface {
	Unpack(ctx context.Context, src string, dst string) error
	FileSuffix() string
}
