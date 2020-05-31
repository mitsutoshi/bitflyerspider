NAME     := bitflyerspider
VERSION  := v1.7.0
REVISION := $(shell git rev-parse --short HEAD)
SRCS     := $(shell find . -type f -name *.go)
LDFLAGS  := -ldflags="-s -w -X \"main.version=$(VERSION)\" -X \"main.revision=$(REVISION)\" -extldflags \"-static\""
S3BUCKET := artifacts-0

## Setup this repository
setup:
	@go get github.com/Songmu/make2help/cmd/make2help
	@go get github.com/golang/dep

## Install dependencies
dep-init: setup
	go mod init
	dep init

## Update dependencies
dep-update: setup
	dep ensure

.PHONY: build
## Build binary
build:
	go build -o bin/$(NAME) $(LDFLAGS)

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
cross-build:
	@for os in linux; do \
		for arch in amd64; do \
			echo "Building for os=$$os arch=$$arch"; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o bin/$$os-$$arch/$(NAME); \
		done; \
	done

.PHONY: deploy
## Deploy binary file.
deploy:
	aws s3 cp bin/linux-amd64/$(NAME) s3://$(S3BUCKET)/bitflyerspider/$(NAME)

