package extract

type Extractor interface {
	// TODO(jan): add ctx
	Unpack(src string, dst string) error
	FileSuffix() string
}
