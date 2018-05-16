VERSION := $(shell git describe --abbrev=0 --tags)
EXACT_VERSION := $(shell git describe --always --long --dirty --tags)
GOPATH := $(GOPATH)
BINARY := rita

LDFLAGS := -ldflags="-X github.com/activecm/rita/config.Version=${VERSION} -X github.com/activecm/rita/config.ExactVersion=${EXACT_VERSION}"
TESTFLAGS := -p=1 -v
# go source files
SRC := $(shell find . -path ./vendor -prune -o -type f -name '*.go' -print)

$(BINARY): vendor
	go build ${LDFLAGS}

.PHONY: install
install: $(BINARY)
	mv $(BINARY) $(GOPATH)/bin/$(BINARY)

.PHONY: test
test: vendor
test: MONGO_ID := $(shell docker run --rm -d mongo:3.6)
test: MONGO_IP := $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(MONGO_ID))
test:
	@until nc -z $(MONGO_IP) 27017; do sleep 1; done; true
	go test $(TESTFLAGS) $(LDFLAGS) ./... -args mongodb://$(MONGO_IP):27017
	@docker stop $(MONGO_ID)

vendor: Gopkg.lock
	dep ensure --vendor-only

Gopkg.lock: $(SRC) Gopkg.toml
	dep ensure --no-vendor
