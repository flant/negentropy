#!/usr/bin/env bash
cd e2e
export ROOT_VAULT_TOKEN=$(cat /tmp/root_token)
export AUTH_VAULT_TOKEN=$(cat /tmp/auth_token)
export ROOT_VAULT_URL=http://127.0.0.1:8300
export AUTH_VAULT_URL=http://127.0.0.1:8200
export ROOT_VAULT_INTERNAL_URL=http://vault_root:8200
export AUTH_VAULT_INTERNAL_URL=http://vault_auth:8200
go mod download
go run github.com/onsi/ginkgo/ginkgo run ./...