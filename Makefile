
COVERDIR = .coverage
TOOLDIR = tools

GO_SRC := $(shell find . -name '*.go' ! -path '*/vendor/*' ! -path 'tools/*' )
GO_DIRS := $(shell find . -type d -name '*.go' ! -path '*/vendor/*' ! -path 'tools/*' )
GO_PKGS := $(shell go list ./... | grep -v '/vendor/')

BINARY = $(shell basename $(shell pwd))
VERSION ?= $(shell git describe --dirty)

CONCURRENT_LINTERS ?= $(shell cat /proc/cpuinfo | grep processor | wc -l)
LINTER_DEADLINE ?= 30s

export PATH := $(TOOLDIR)/bin:$(PATH)
SHELL := env PATH=$(PATH) /bin/bash

all: style lint test $(BINARY).x86_64

$(BINARY).x86_64: $(GO_SRC)
	CGO_ENABLED=0 go build -a -ldflags "-extldflags '-static' -X main.Version=$(VERSION)" -o $(BINARY).x86_64 .

style: tools
	gometalinter --disable-all --enable=gofmt --vendor

lint: tools
	@echo Using $(CONCURRENT_LINTERS) processes
	gometalinter -j $(CONCURRENT_LINTERS) --deadline=$(LINTER_DEADLINE) --disable=gotype $(GO_DIRS)

fmt: tools
	gofmt -s -w $(GO_SRC)

test: tools
	@mkdir -p $(COVERDIR)
	@rm -f $(COVERDIR)/*
	for pkg in $(GO_PKGS) ; do \
		go test -v -covermode count -coverprofile=$(COVERDIR)/$$(echo $$pkg | tr '/' '-').out $$pkg ; \
	done
	gocovmerge $(shell find $(COVERDIR) -name '*.out') > cover.out

tools:
	$(MAKE) -C $(TOOLDIR)

.PHONY: tools style fmt test all
