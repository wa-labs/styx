FROM golang:1.7-alpine

ENV SRC_PATH="/go/src/github.com/wa-labs/styx"

RUN apk add --update git \
&& mkdir -p $SRC_PATH
COPY . $SRC_PATH
WORKDIR $SRC_PATH

RUN dep ensure -v \
&& go build -v \
&& cp styx /usr/local/bin \
&& mkdir -p /app \
&& touch /app/config.yml \
&& apk del git \
&& rm -rf /go/*

WORKDIR /

ENV CONFIG_FILE=/app/config.yml
EXPOSE 8080 8081 8082
ENTRYPOINT ["styx"]
