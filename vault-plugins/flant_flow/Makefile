# to use start enable openapi - needs installed vault
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

.PHONY: all
all: fmt build start openapi

.PHONY: build
build:
	CGO_ENABLED=1 GOOS="$(OS)" GOARCH="$(GOARCH)" go build -o ../build/flant_flow cmd/flant_flow/main.go

.PHONY: build
build-linux:
	CGO_ENABLED=1 GOOS="linux" GOARCH="$(GOARCH)" go build -o ../build/flant_flow cmd/flant_flow/main.go

.PHONY: start
start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=../build

.PHONY: enable
enable:
	export VAULT_ADDR=http://0.0.0.0:8200 && export VAULT_TOKEN=root && vault secrets enable -path=flant_flow flant_flow

.PHONY: clean
clean:
	rm -f ../build/flant_flow

.PHONY: fmt
fmt:
	-@$$(go env GOPATH)/bin/golangci-lint run ./... --config ../.golangci.yaml --fix

.PHONY: deps
deps:
	go mod tidy
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.38.0

.PHONY: openapi
openapi:
	curl -s -H "X-Vault-Token: root" "http://127.0.0.1:8200/v1/sys/internal/specs/openapi" | jq > openapi.json


.PHONY: test
test:
	go test -race ./...

.PHONY: precommit
precommit: fmt build test openapi
