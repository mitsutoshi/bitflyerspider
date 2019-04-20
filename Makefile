NAME     := bitflyerspider
VERSION  := v1.1.0
REVISION := $(shell git rev-parse --short HEAD)
SRCS     := $(shell find . -type f -name *.go)
LDFLAGS  := -ldflags="-s -w -X \"main.version=$(VERSION)\" -X \"main.revision=$(REVISION)\" -extldflags \"-static\""

## Setup this repository
setup:
	@go get github.com/Songmu/make2help/cmd/make2help
	@go get github.com/golang/dep

## Install dependencies
dep-init: setup
	dep init

## Update dependencies
dep-update: setup
	dep ensure

## Build binary
.PHONY: build
bin/$(NAME):
	go build -o bin/$(VERSION)/$(NAME) -ldflags "$(LDFLAGS)"

## Show help
help:
	@make2help $(MAKEFILE_LIST)

.PHONY: clean
## Clean
clean:
	rm -rf bin/*
	rm -rf vendor/*

.PHONY: cross-build
## Cross-build
cross-build: dep-update
	@for os in linux; do \
		for arch in amd64 386; do \
			echo "Building for os=$$os arch=$$arch"; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o bin/$(VERSION)/$$os-$$arch/$(NAME); \
		done; \
	done
