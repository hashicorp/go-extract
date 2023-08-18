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
	@cd ./cmd/extract && go build -o extract .
	@mv ./cmd/extract/extract extract

install: build
	@mv extract $(GOPATH)/bin/extract

clean:
	@go clean
	@rm extract

test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

all: build install
