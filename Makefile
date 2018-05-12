VERSION := $(shell git describe --abbrev=0 --tags)
EXACT_VERSION := $(shell git describe --always --long --dirty --tags)
GOPATH := $(GOPATH)
BINARY := rita

LDFLAGS=-ldflags="-X github.com/activecm/rita/config.Version=${VERSION} -X github.com/activecm/rita/config.ExactVersion=${EXACT_VERSION}"


default:
	dep ensure
	go build ${LDFLAGS}

install:
	dep ensure
	go build ${LDFLAGS} -o ${GOPATH}/bin/${BINARY}

test:
	dep ensure
	go test ${LDFLAGS} ./...
