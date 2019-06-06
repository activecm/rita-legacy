FROM golang:1.10-alpine as rita-builder

RUN apk add --no-cache git make ca-certificates wget build-base
RUN wget -q -O /go/bin/dep https://github.com/golang/dep/releases/download/v0.5.3/dep-linux-amd64 && chmod +x /go/bin/dep

WORKDIR /go/src/github.com/activecm/rita

# cache dependencies
COPY Gopkg.lock Gopkg.toml Makefile ./
RUN make vendor

# copy the rest of the code
COPY . ./

# Change ARGs with --build-arg to target other architectures
# Produce a self-contained statically linked binary
ARG CGO_ENABLED=0
# Set the build target architecture and OS
ARG GOARCH=amd64
ARG GOOS=linux
# Passing arguments in to make result in them being set as
# environment variables for the call to go build
RUN make CGO_ENABLED=$CGO_ENABLED GOARCH=$GOARCH GOOS=$GOOS

FROM scratch

WORKDIR /
COPY --from=rita-builder /go/src/github.com/activecm/rita/etc/rita.yaml /etc/rita/config.yaml
COPY --from=rita-builder /go/src/github.com/activecm/rita/rita /rita

ENTRYPOINT ["/rita"]
