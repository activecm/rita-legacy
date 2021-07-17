VERSION := $(shell git describe --abbrev=0 --tags)
EXACT_VERSION := $(shell git describe --always --long --dirty --tags)
PREFIX ?= /usr/local

LDFLAGS := -ldflags='-X github.com/activecm/rita/config.Version=${VERSION} -X github.com/activecm/rita/config.ExactVersion=${EXACT_VERSION}'
TESTFLAGS := -p=1 -v
# go source files
SRC := $(shell find . -path ./vendor -prune -o -type f -name '*.go' -print)

# Allow a variable to be initialized and cached on first use. Subsequent uses will
# use the cached value instead of evaluating the variable's declaration again.
# Use like this: VAR = $(call cache,VAR)
# https://www.cmcrossroads.com/article/makefile-optimization-eval-and-macro-caching
cache = $(if $(cached-$1),,$(eval cached-$1 := 1)$(eval cache-$1 := $($1)))$(cache-$1)

# force rita to be rebuilt even if it's up to date
.PHONY: rita
rita: $(SRC)
	@# remove any existing vendor directory from dep
	@rm -rf vendor
	go build ${LDFLAGS}

.PHONY: install
install: rita
	mv rita $(PREFIX)/bin/
	mkdir -p $(PREFIX)/etc/bash_completion.d/ $(PREFIX)/etc/rita/
	sudo cp etc/bash_completion.d/rita $(PREFIX)/etc/bash_completion.d/rita
	sudo cp etc/rita.yaml $(PREFIX)/etc/rita/config.yaml

.PHONY: docker-check
# Use this recipe if you want to fail if docker is missing
docker-check:
	@if ! docker ps > /dev/null; then echo "Ensure docker is installed and accessible from the current user context"; return 1; fi

.PHONY: integration-test
integration-test: docker-check
# docker run should only get executed once on initialization using the cache trick
integration-test: MONGO_EXE = $(shell docker run --rm -d mongo:4.2)
integration-test: MONGO_ID = $(call cache,MONGO_EXE)
integration-test: MONGO_IP = $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(MONGO_ID))
integration-test:
	@echo Waiting for Mongo to respond to connection attempts
	@until nc -z $(MONGO_IP) 27017; do sleep 1; done; true
	@echo Running tests
	@bash -c "trap 'docker stop $(MONGO_ID) > /dev/null' EXIT; go test $(TESTFLAGS) -tags=integration $(LDFLAGS) ./... -args mongodb://$(MONGO_IP):27017"


.PHONY: test
test: static-test unit-test
	@echo Ran tests on rita $(EXACT_VERSION)

.PHONY: static-test
static-test:
	golangci-lint run ./...

.PHONY: unit-test
unit-test:
	go test -race -cover $(shell go list ./... | grep -v /vendor/)


# The following targets all use docker

.PHONY: docker-build
docker-build:
	docker build -t quay.io/activecm/rita:latest -f Dockerfile .

.PHONY: docker-build-test
docker-build-test:
	docker build -t quay.io/activecm/rita:test -f test.Dockerfile .

# Runs all tests inside docker container
.PHONY: docker-test
docker-test: docker-build-test
	docker run --rm quay.io/activecm/rita:test make test

.PHONY: docker-unit-test
docker-unit-test: docker-build-test
	docker run --rm quay.io/activecm/rita:test make unit-test

.PHONY: docker-static-test
docker-static-test: docker-build-test
	docker run --rm quay.io/activecm/rita:test make static-test

# .PHONY: docker-integration-test
# docker-integration-test: docker-build-test
# 	docker run --rm quay.io/activecm/rita:test make integration-test
