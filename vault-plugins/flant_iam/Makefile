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

all: fmt build start

build:
	GOOS="$(OS)" GOARCH="$(GOARCH)" go build -o build/flant_iam cmd/flant_iam/main.go


build-linux:
	GOOS="linux" GOARCH="$(GOARCH)" go build -o build/flant_iam cmd/flant_iam/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./build

enable:
	vault secrets enable -path=mock-secrets flant_iam

clean:
	rm -f ./build/flant_iam

fmt:
	@$$(go env GOPATH)/bin/golangci-lint run ./... -E gofumpt --fix
	pushd tests && yarn run prettier --write ./**.js && popd

deps:
	go mod tidy
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.38.0
	pushd tests && yarn && popd

test:
	go test -race ./...
	pushd tests && yarn && yarn test

.PHONY: build build-linux deps clean fmt start enable test
