#!/bin/bash

set -e

cd e2e

vault_counts=$(docker ps --format '{{.Names}}' | grep vault | wc -l)
if (( $vault_counts < 2 )); then
  echo "DEBUG: found single vault instance"
  export ROOT_VAULT_TOKEN=$(cat /tmp/vault_single_token)
  export AUTH_VAULT_TOKEN=$(cat /tmp/vault_single_token)
  export ROOT_VAULT_URL=https://localhost:8200
  export AUTH_VAULT_URL=https://localhost:8200
  export ROOT_VAULT_INTERNAL_URL=https://vault:8200
  export AUTH_VAULT_INTERNAL_URL=https://vault:8200
else
  echo "DEBUG: found two separate vaults"
  export ROOT_VAULT_TOKEN=$(cat /tmp/vault_root_token)
  export AUTH_VAULT_TOKEN=$(cat /tmp/vault_auth_token)
  export ROOT_VAULT_URL=https://localhost:8300
  export AUTH_VAULT_URL=https://localhost:8200
  export ROOT_VAULT_INTERNAL_URL=https://vault-root:8300
  export AUTH_VAULT_INTERNAL_URL=https://vault-auth:8200
fi

echo DEBUG: ROOT_VAULT_TOKEN is $ROOT_VAULT_TOKEN
echo DEBUG: AUTH_VAULT_TOKEN is $AUTH_VAULT_TOKEN
echo DEBUG: ROOT_VAULT_URL is $ROOT_VAULT_URL
echo DEBUG: AUTH_VAULT_URL is $AUTH_VAULT_URL
echo DEBUG: ROOT_VAULT_INTERNAL_URL is $ROOT_VAULT_INTERNAL_URL
echo DEBUG: AUTH_VAULT_INTERNAL_URL is $AUTH_VAULT_INTERNAL_URL

go mod download

# run watcher
go run tests/lib/item_watcher/watcher.go &> /dev/null &

go run github.com/onsi/ginkgo/ginkgo run ./...

# watcher output & shutdown
echo
echo items:
curl localhost:3333/report
curl localhost:3333/shutdown