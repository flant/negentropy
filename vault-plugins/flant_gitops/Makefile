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
	VAULT_ADDR='http://127.0.0.1:8200' vault auth enable -ca-cert=examples/conf/ca-cert.pem approle
	VAULT_ADDR='http://127.0.0.1:8200' vault policy write -ca-cert=examples/conf/ca-cert.pem good examples/conf/good.hcl
	VAULT_ADDR='http://127.0.0.1:8200' VAULT_ADDR='http://localhost:8200' vault write -ca-cert=examples/conf/ca-cert.pem auth/approle/role/good secret_id_ttl=30m token_ttl=90s token_policies=good
	VAULT_ADDR='http://127.0.0.1:8200' vault write -ca-cert=examples/conf/ca-cert.pem flant-gitops/configure_vault_access \
                vault_addr="http://localhost:8200" \
                vault_tls_server_name="localhost" \
		role_name="good" \
		secret_id_ttl="120m" \
		approle_mount_point="auth/approle" \
                secret_id="$$(VAULT_ADDR='http://localhost:8200' vault write -ca-cert=examples/conf/ca-cert.pem -format=json -f auth/approle/role/good/secret-id | jq -r '.data.secret_id')" \
                role_id="$$(VAULT_ADDR='http://localhost:8200' vault read -ca-cert=examples/conf/ca-cert.pem -format=json auth/approle/role/good/role-id | jq -r '.data.role_id')" \
                vault_cacert="$$(cat examples/conf/ca-cert.pem)"
	VAULT_ADDR='http://127.0.0.1:8200' vault write flant-gitops/configure git_repo_url=https://github.com/flant/negentropy git_branch_name=flant_gitops_test_infra required_number_of_verified_signatures_on_commit=0 git_poll_period="1m" initial_last_successful_commit="" docker_image="alpine:3.14.0@sha256:234cb88d3020898631af0ccbbcca9a66ae7306ecd30c9720690858c1b007d2a0" command="./flant_gitops.sh"
	VAULT_ADDR='http://127.0.0.1:8200' vault write flant-gitops/configure/vault_request/get_configuration name=get_configuration method=GET path="v1/flant-gitops/configure" wrap_ttl="1m"

	touch flant_gitops.log
	tail -f flant_gitops.log

clean:
	rm -f ./vault/plugins/vault-plugin-flant-gitops

fmt:
	go fmt $$(go list ./...)
	gci -local github.com/flant/negentropy

.PHONY: build clean fmt start enable
