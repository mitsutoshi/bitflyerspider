NAME     := bitflyerspider
VERSION  := v1.0.0
REVISION := $(shell git rev-parse --short HEAD)
SRCS     := $(shell find . -type f -name *.go)
LDFLAGS  := -X 'main.version=$(VERSION)' -X 'main.revision=$(REVISION)'

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
bin/$(NAME):
	go build -o bin/$(NAME) -ldflags "$(LDFLAGS)"

## Show help.
help:
	@make2help $(MAKEFILE_LIST)
