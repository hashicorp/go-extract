default: build

################################################################
# identify deps for installation
################################################################
# Start with default
# UNAME := $(shell uname)
# INSTALL_DEPS="sudo apt update && sudo apt install wkhtmltopdf fontforge woff2"
# ifeq ($(UNAME), Darwin) 
# INSTALL_DEPS="brew install wkhtmltopdf"
# endif

build:
	@cd ./cmd/goextract && go build -o goextract .
	@mv ./cmd/goextract/goextract goextract

install: build
	@mv goextract $(GOPATH)/bin/goextract

clean:
	@go clean
	@rm goextract

test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

test_coverage_view:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

test_coverage_html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o=coverage.html

fuzz:
	# Fuzzing FuzzSecurityCheckOs in ./extractor/target_test.go
	go test ./internal/extractor -run=FuzzSecurityCheckOs -fuzz=FuzzSecurityCheckOs -fuzztime=30s
	# Fuzzing FuzzDetermineOutputName in ./extractor/decompress_test.go
	go test ./internal/extractor -run=FuzzDetermineOutputName -fuzz=FuzzDetermineOutputName -fuzztime=30s

all: build install



# Create fuzzing entries with following script:
# #!/bin/bash
#
# set -e
#
# fuzzTime=${1:-30}
#
# files=$(grep -r --include='**_test.go' --files-with-matches 'func Fuzz' .)
#
# for file in ${files}
# do
#     funcs=$(grep -o 'func Fuzz\w*' $file | sed 's/func //')
#     for func in ${funcs}
#     do
#         echo "# Fuzzing $func in $file"
#         parentDir=$(dirname $file)
#         echo "go test $parentDir -run=$func -fuzz=$func -fuzztime=${fuzzTime}s"
#     done
# done
