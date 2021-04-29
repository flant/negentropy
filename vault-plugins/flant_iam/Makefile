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

all: fmt build start openapi

build:
	mkdir -p build
	CGO_ENABLED=0 GOOS="$(OS)" GOARCH="$(GOARCH)" go build -o build/vault-plugin-flant-iam cmd/flant_iam/main.go


build-linux:
	mkdir -p build
	CGO_ENABLED=0 GOOS="linux" GOARCH="$(GOARCH)" go build -o build/vault-plugin-flant-iam cmd/flant_iam/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./build

enable:
	vault secrets enable -path=mock-secrets flant_iam

clean:
	rm -f ./build/flant_iam

fmt:
	-$$(go env GOPATH)/bin/golangci-lint run ./... -E gofumpt --fix --disable-all
	-@pushd tests && yarn run prettier --write ./tests/**.mjs && popd

deps:
	go mod tidy
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.38.0
	pushd tests && yarn && popd

openapi:
	curl -s -H "X-Vault-Token: root" "http://127.0.0.1:8200/v1/sys/internal/specs/openapi" | jq > openapi.json

test:
	go test -race ./...
	pushd tests && yarn && yarn test

precommit: fmt build test openapi


.PHONY: build build-linux deps clean fmt start enable test
