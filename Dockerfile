FROM golang:1.17-alpine3.15 AS builder

RUN apk add --update --no-cache make bash git openssh-client build-base musl-dev curl wget

ADD . /src/app

WORKDIR /src/app

RUN export CGO_ENABLED=0 && \
    export GO111MODULE=on && \
    go build -o ./bin/app -ldflags '-s -w' cmd/smtp/main.go

FROM alpine:3.15

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
COPY --from=builder /src/app/bin/app /app
COPY --from=builder /src/app/configs/config.yaml  /configs/config.yaml


ENTRYPOINT ["/app"]
