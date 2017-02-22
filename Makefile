PKG := github.com/ocmdev/rita
VERSION := $(shell git describe --always --long --dirty)

default:
	go get
	go build -ldflags="-X github.com/ocmdev/rita/config.VERSION=${VERSION}"

install:
	go install
