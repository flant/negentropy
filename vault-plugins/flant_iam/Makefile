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
	mkdir -p build
	CGO_ENABLED=0 GOOS="$(OS)" GOARCH="$(GOARCH)" go build -o build/vault-plugin-flant-iam cmd/flant_iam/main.go


.PHONY: build
build-linux:
	mkdir -p build
	CGO_ENABLED=0 GOOS="linux" GOARCH="$(GOARCH)" go build -o build/vault-plugin-flant-iam cmd/flant_iam/main.go

.PHONY: start
start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./build

.PHONY: enable
enable:
	vault secrets enable -path=mock-secrets flant_iam

.PHONY: clean
clean:
	rm -f ./build/flant_iam

.PHONY: fmt
fmt: generate
	-@$$(go env GOPATH)/bin/golangci-lint run ./... --config ../.golangci.yaml --fix

.PHONY: deps
deps:
	go mod tidy
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.38.0

.PHONY: openapi
openapi:
	curl -s -H "X-Vault-Token: root" "http://127.0.0.1:8200/v1/sys/internal/specs/openapi" | jq > openapi.json

# Somewhere there should be done
# docker exec -i $(docker ps | grep vault_1 | cut -f1 -d" ") sh -c 'vault token create -orphan -policy=root -field=token > /vault/testdata/token'
# docker exec -i $(docker ps | grep vault_1 | cut -f1 -d" ") sh -c 'vault plugin register -sha256=$(sha256sum /vault/plugins/flant_iam | cut -d' ' -f1) secret flant_iam  '
# docker exec -i $(docker ps | grep vault_1 | cut -f1 -d" ") sh -c 'vault secrets enable -path=flant_iam flant_iam'
.PHONY: test
test:
	go test -race ./...

.PHONY: generate
generate:
	@go generate ./...

.PHONY: precommit
precommit: fmt build test openapi
