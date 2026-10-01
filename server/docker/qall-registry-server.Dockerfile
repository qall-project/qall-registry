FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY pkg pkg
COPY internal internal
COPY Makefile .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN make build

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

RUN adduser -D -g '' qall

COPY --from=builder /app/github.com/qall-project/qall-registry/server /usr/local/bin/github.com/qall-project/qall-registry/server

RUN mkdir -p /.data && chown -R qall:qall /.data

WORKDIR /.data
USER qall

ENTRYPOINT ["/usr/local/bin/github.com/qall-project/qall-registry/server"]