GOARCH = amd64

UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: build start

build:
	GOOS="$(OS)" GOARCH="$(GOARCH)" go build -mod=mod -o bin/vault-plugin-flant_iam_auth cmd/vault-plugin-flant-iam-auth/main.go


build-linux:
	GOOS="linux" GOARCH="$(GOARCH)" go build -mod=mod -o bin/vault-plugin-flant_iam_auth cmd/vault-plugin-flant-iam-auth/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./bin

enable:
	vault secrets enable -path=mock-secrets flant_iam_auth

clean:
	rm -f ./bin/vault-plugin-flant_iam_auth

fmt:
	-@$$(go env GOPATH)/bin/golangci-lint run ./... -E gofumpt --fix

deps:
	go mod tidy
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.38.0

test:
	go test -race ./...
	pushd tests && yarn && yarn test

.PHONY: build build-linux deps clean fmt start enable test