#!/bin/bash

set -e

cd e2e

vault_counts=$(docker ps --format '{{.Names}}' | grep vault | wc -l)
if (( $vault_counts < 2 )); then
  echo "DEBUG: found single vault instance"
  export ROOT_VAULT_TOKEN=$(cat /tmp/vault_dev_token)
  export AUTH_VAULT_TOKEN=$(cat /tmp/vault_dev_token)
  export ROOT_VAULT_URL=http://127.0.0.1:8200
  export AUTH_VAULT_URL=http://127.0.0.1:8200
  export ROOT_VAULT_INTERNAL_URL=http://vault:8200
  export AUTH_VAULT_INTERNAL_URL=http://vault:8200
else
  echo "DEBUG: found two separate vaults"
  export ROOT_VAULT_TOKEN=$(cat /tmp/root_token)
  export AUTH_VAULT_TOKEN=$(cat /tmp/auth_token)
  export ROOT_VAULT_URL=http://127.0.0.1:8300
  export AUTH_VAULT_URL=http://127.0.0.1:8200
  export ROOT_VAULT_INTERNAL_URL=http://vault-root:8200
  export AUTH_VAULT_INTERNAL_URL=http://vault-auth:8200
fi

echo DEBUG: ROOT_VAULT_TOKEN is $ROOT_VAULT_TOKEN
echo DEBUG: AUTH_VAULT_TOKEN is $AUTH_VAULT_TOKEN
echo DEBUG: ROOT_VAULT_URL is $ROOT_VAULT_URL
echo DEBUG: AUTH_VAULT_URL is $AUTH_VAULT_URL
echo DEBUG: ROOT_VAULT_INTERNAL_URL is $ROOT_VAULT_INTERNAL_URL
echo DEBUG: AUTH_VAULT_INTERNAL_URL is $AUTH_VAULT_INTERNAL_URL

go mod download
go run tests/restoration_speed/test.go
go run github.com/onsi/ginkgo/ginkgo run ./...
