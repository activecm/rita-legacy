# use debian instead of alpine because the go race requires glibc
# https://github.com/golang/go/issues/14481
FROM golang:1.10

RUN apt-get update && apt-get install -y git make ca-certificates wget build-essential
WORKDIR /go
# install dep for vendoring
RUN wget -q -O bin/dep https://github.com/golang/dep/releases/download/v0.5.3/dep-linux-amd64 \
	&& chmod +x bin/dep \
# install testing dependencies
	&& wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh \
	| sh -s v1.16.0

WORKDIR /go/src/github.com/activecm/rita

# cache dependencies
COPY Gopkg.lock Gopkg.toml Makefile ./
RUN make vendor

# copy the rest of the code
COPY . ./

CMD ["make", "test"]
