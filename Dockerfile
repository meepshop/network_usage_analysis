FROM alpine:latest

WORKDIR /opt/app

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

RUN update-ca-certificates

COPY main /opt/app