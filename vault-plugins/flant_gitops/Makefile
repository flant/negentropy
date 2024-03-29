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
	vault server -dev -dev-listen-address 0.0.0.0:8200 -dev-root-token-id=root -dev-plugin-dir=./vault/plugins

enable: export VAULT_ADDR = http://127.0.0.1:8200
enable:
	vault secrets enable -path=flant-gitops vault-plugin-flant-gitops
	vault auth enable -ca-cert=examples/conf/ca-cert.pem approle
	vault policy write -ca-cert=examples/conf/ca-cert.pem good examples/conf/good.hcl
	vault write -ca-cert=examples/conf/ca-cert.pem auth/approle/role/good secret_id_ttl=30m token_ttl=90s token_policies=good
	vault write -ca-cert=examples/conf/ca-cert.pem flant-gitops/configure_vault_access \
		vault_addr="http://$$(docker network inspect bridge -f '{{ (index .IPAM.Config 0).Gateway }}'):8200" \
		vault_tls_server_name="localhost" \
		role_name="good" \
		secret_id_ttl="120m" \
		approle_mount_point="auth/approle" \
		secret_id="$$(vault write -ca-cert=examples/conf/ca-cert.pem -format=json -f auth/approle/role/good/secret-id | jq -r '.data.secret_id')" \
		role_id="$$(vault read -ca-cert=examples/conf/ca-cert.pem -format=json auth/approle/role/good/role-id | jq -r '.data.role_id')" \
		vault_cacert="$$(cat examples/conf/ca-cert.pem)"
	vault write flant-gitops/configure git_repo_url=https://github.com/flant/negentropy git_branch_name=flant_gitops_test_infra required_number_of_verified_signatures_on_commit=0 git_poll_period="1m" initial_last_successful_commit="" docker_image="vault:1.7.3@sha256:53e509aaa6f72c54418b2f65f23fdd8a5ddd22bf6521c4b5bf82a8ae4edd0e53" commands="./flant_gitops.sh"
	vault write flant-gitops/configure/vault_request/get_configuration method=GET path="/v1/flant-gitops/configure" wrap_ttl="1m" @payload-example.json

	vault read flant-gitops/configure
	vault read flant-gitops/configure/vault_request/get_configuration

	touch flant_gitops.log
	tail -f flant_gitops.log

clean:
	rm -f ./vault/plugins/vault-plugin-flant-gitops

fmt:
	go fmt $$(go list ./...)
	gci -local github.com/flant/negentropy

.PHONY: build clean fmt start enable
