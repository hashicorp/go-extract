module fetch-testing

go 1.21.0

require (
	github.com/alecthomas/kong v0.9.0
	github.com/hashicorp/go-extract v0.3.2-0.20240313145657-056868eb167a
	github.com/hashicorp/go-slug v0.14.0
	github.com/pkg/errors v0.9.1
	golang.org/x/sync v0.6.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	golang.org/x/sys v0.17.0 // indirect
)

replace github.com/hashicorp/go-extract => ./..
