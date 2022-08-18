# use debian instead of alpine because the go race requires glibc
# https://github.com/golang/go/issues/14481
FROM golang:1.17

RUN apt-get update && apt-get install -y git make ca-certificates wget build-essential
WORKDIR /go
# install testing dependencies
RUN wget -O - -q https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
	| sh -s v1.41.1

WORKDIR /go/src/github.com/activecm/rita

# cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy the rest of the code
COPY . ./

CMD ["make", "test"]
