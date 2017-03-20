# Makefile to build the tools used in the build system.
GO_SRC := $(shell find $(SOURCEDIR) -name '*.go')

all: gocovmerge gometalinter

gocovmerge: $(GO_SRC)
	go build -o gocovmerge ./vendor/github.com/wadey/gocovmerge

gometalinter: $(GO_SRC)
	go build -o gometalinter ./vendor/github.com/alecthomas/gometalinter
