VERSION := $(shell git describe --abbrev=0 --tags)
EXACT_VERSION := $(shell git describe --always --long --dirty --tags)
GOPATH := $(GOPATH)
BINARY := rita

LDFLAGS=-ldflags="-X github.com/activecm/rita/config.Version=${VERSION} -X github.com/activecm/rita/config.ExactVersion=${EXACT_VERSION}"

# go source files
SRC = $(shell find . -type f -name '*.go')

$(BINARY): $(SRC) vendor
	go build ${LDFLAGS}

.PHONY: install
install: vendor
	go build ${LDFLAGS} -o ${GOPATH}/bin/${BINARY}

.PHONY: test
test: vendor
	go test ${LDFLAGS} ./...

vendor: Gopkg.toml
	dep ensure
