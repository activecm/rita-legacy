VERSION := $(shell git describe --always --long --dirty)
GOPATH := $(GOPATH)
BINARY := rita

LDFLAGS=-ldflags="-X github.com/ocmdev/rita/config.VERSION=${VERSION}"


default:
	go get
	go build ${LDFLAGS}

# Having issues with 'go install' + LDFLAGS using sudo and the
# install script. This is a workaround.
install:
	go get
	go build ${LDFLAGS} -o ${GOPATH}/bin/${BINARY}

