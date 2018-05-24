VERSION := $(shell git describe --abbrev=0 --tags)
EXACT_VERSION := $(shell git describe --always --long --dirty --tags)
GOPATH := $(GOPATH)
BINARY := rita

LDFLAGS := -ldflags='-X github.com/activecm/rita/config.Version=${VERSION} -X github.com/activecm/rita/config.ExactVersion=${EXACT_VERSION}'
TESTFLAGS := -p=1 -v
# go source files
SRC := $(shell find . -path ./vendor -prune -o -type f -name '*.go' -print)

# https://www.cmcrossroads.com/article/makefile-optimization-eval-and-macro-caching
cache = $(if $(cached-$1),,$(eval cached-$1 := 1)$(eval cache-$1 := $($1)))$(cache-$1)

$(BINARY): vendor
	go build ${LDFLAGS}

.PHONY: install
install: $(BINARY)
	mv $(BINARY) $(GOPATH)/bin/$(BINARY)

.PHONY: test
test: vendor
test: MONGO_EXE = $(shell docker run --rm -d mongo:3.6)
test: MONGO_ID = $(call cache,MONGO_EXE)
test: MONGO_IP = $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(MONGO_ID))
test:
	@until nc -z $(MONGO_IP) 27017; do sleep 1; done; true
	@bash -c "trap 'docker stop $(MONGO_ID) > /dev/null' EXIT; go test $(TESTFLAGS) $(LDFLAGS) ./... -args mongodb://$(MONGO_IP):27017"

vendor: Gopkg.lock
	dep ensure --vendor-only

Gopkg.lock: $(SRC) Gopkg.toml
	dep ensure --no-vendor
