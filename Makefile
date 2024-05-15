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
	go test ./extractor -run=FuzzDetermineOutputName -fuzz=FuzzDetermineOutputName -fuzztime=30s

all: build install
