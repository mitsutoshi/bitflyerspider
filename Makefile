NAME     := bitflyerspider
VERSION  := v1.0.2
REVISION := $(shell git rev-parse --short HEAD)
SRCS     := $(shell find . -type f -name *.go)
LDFLAGS  := -ldflags="-s -w -X \"main.version=$(VERSION)\" -X \"main.revision=$(REVISION)\" -extldflags \"-static\""

## Setup this repository.
setup:
	@go get github.com/Songmu/make2help/cmd/make2help
	@go get github.com/golang/dep

## Install dependencies.
dep-init: setup
	dep init

## Update dependencies.
dep-update: setup
	dep ensure

## Build binary
.PHONY: bin/$(NAME)
bin/$(NAME):
	go build -o bin/$(VERSION)/$(NAME) -ldflags "$(LDFLAGS)"

## Show help.
help:
	@make2help $(MAKEFILE_LIST)

.PHONY: clean
clean:
	rm -rf bin/*
	rm -rf vendor/*

.PHONY: cross-build
cross-build: dep-update
	@for os in darwin linux windows; do \
		for arch in amd64 386; do \
			echo "Building for os=$$os arch=$$arch"; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o bin/$(VERSION)/$$os-$$arch/$(NAME); \
		done; \
	done
