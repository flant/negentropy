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

build_and_start: builddir build start

test: enable_approle create_policy create_role enable_plugin get_config_no_tls write_access_conf getting_role reinit_client get_config

builddir:
	mkdir -p bin/

build:
	GOOS="$(OS)" GOARCH="$(GOARCH)" go build -mod=mod -o bin/example-plugin cmd/main.go

build-linux:
	GOOS="linux" GOARCH="$(GOARCH)" go build -mod=mod -o bin/example-plugin cmd/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./bin -config=conf/vault.hcl

enable_approle:
	VAULT_ADDR='http://localhost:8200' vault auth enable -ca-cert=conf/ca-cert.pem  approle
create_policy:
	VAULT_ADDR='http://localhost:8200' vault policy write -ca-cert=conf/ca-cert.pem good good.hcl
create_role:
	VAULT_ADDR='http://localhost:8200' vault write -ca-cert=conf/ca-cert.pem auth/approle/role/good secret_id_ttl=30m token_ttl=90s token_policies=good
enable_plugin:
	VAULT_ADDR='http://localhost:8200' vault auth enable -ca-cert=conf/ca-cert.pem -path=aa example-plugin
write_access_conf:
	VAULT_ADDR='http://localhost:8200' vault write -ca-cert=conf/ca-cert.pem auth/aa/configure_vault_access \
		vault_addr="http://localhost:8200" \
		vault_tls_server_name="vault_host" \
		role_name="good" \
		secret_id_ttl="120m" \
		approle_mount_point="/auth/approle/" \
		secret_id="$$(VAULT_ADDR='http://localhost:8200' vault write -ca-cert=conf/ca-cert.pem -format=json -f auth/approle/role/good/secret-id | jq -r '.data.secret_id')" \
		role_id="$$(VAULT_ADDR='http://localhost:8200' vault read -ca-cert=conf/ca-cert.pem -format=json auth/approle/role/good/role-id | jq -r '.data.role_id')" \
		vault_cacert="$$(cat conf/ca-cert.pem)"
getting_role:
	VAULT_ADDR='http://localhost:8200' vault read -ca-cert=conf/ca-cert.pem auth/aa/read_role

get_config:
	VAULT_ADDR='http://localhost:8200' vault read -ca-cert=conf/ca-cert.pem auth/aa/get_conf

get_config_no_tls:
	VAULT_ADDR='http://localhost:8200' vault read -tls-skip-verify auth/aa/get_conf

reinit_client:
	VAULT_ADDR='http://localhost:8200' vault write -force -ca-cert=conf/ca-cert.pem auth/aa/read_role

clean:
	rm -f ./bin/example-plugin

.PHONY: build build-linux clean start test
