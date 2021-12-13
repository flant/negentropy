# syntax=docker/dockerfile:1
FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./mock ./mock

RUN go build -o /oidc ./cmd/server.go

EXPOSE 9998

CMD ["/oidc"]
