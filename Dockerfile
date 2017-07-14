#RITA runs in Docker!
#However, it needs a little help.
#In order to run rita in Docker, two volume mounts are needed. 
#One for logs, and another for the config file.
#Alternatively you may extend this dockerfile and add in these files.
#Make sure your Dockerized RITA config file points to the correct bro log location.
#Additionally, make sure that RITA has access to a MongoDB server.

#Ex: docker run -it --rm -v /path/to/bro/logs:/logs/:ro -v /path/to/rita/config.yaml:/root/.rita/config.yaml:ro rita import
#RITA works best with docker-compose. Docker-compose lets you set these mounts
#and additionally connect it to MongoDB with ease.
FROM golang:1.8-alpine as rita-builder
RUN apk update && apk upgrade && apk add --no-cache git && apk add --no-cache make && apk add --no-cache ca-certificates
RUN mkdir /logs
RUN mkdir $HOME/.rita/
WORKDIR /go/src/github.com/ocmdev/rita
COPY . .
RUN make

FROM alpine:latest
COPY --from=rita-builder /go/src/github.com/ocmdev/rita/rita .
ENTRYPOINT ["./rita"]
