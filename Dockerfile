FROM golang:1.12.4-alpine3.9 AS builder

RUN apk add --update --no-cache make bash git openssh-client build-base musl-dev curl wget

ADD . /src/app

WORKDIR /src/app

RUN mkdir ./bin && \
    go build -o ./bin/app -a main.go

FROM alpine:3.9

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
COPY --from=builder /src/app/bin/app /app
COPY ./config/config.yaml /config/config.yaml


ENTRYPOINT ["/app"]