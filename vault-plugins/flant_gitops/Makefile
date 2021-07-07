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

lint:
	GOOS=$(OS) GOARCH="$(GOARCH)" golangci-lint run

build:
	GOOS=$(OS) GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-flant-gitops cmd/flant_gitops/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./vault/plugins -log-level trace

enable:
	VAULT_ADDR='http://127.0.0.1:8200' vault secrets enable -path=flant-gitops vault-plugin-flant-gitops
	VAULT_ADDR='http://127.0.0.1:8200' vault write flant-gitops/configure git_repo_url=https://github.com/flant/negentropy git_branch_name=flant_gitops_test_infra required_number_of_verified_signatures_on_commit=0 git_poll_period=1 last_successful_commit="" docker_image="alpine:3.14.0@sha256:234cb88d3020898631af0ccbbcca9a66ae7306ecd30c9720690858c1b007d2a0" command="flant_gitops.sh"

	touch flant_gitops.log
	tail -f flant_gitops.log

clean:
	rm -f ./vault/plugins/vault-plugin-flant-gitops

fmt:
	go fmt $$(go list ./...)
	gci -local github.com/flant/negentropy

.PHONY: build clean fmt start enable
