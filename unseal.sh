#!/usr/bin/env bash

set -e

function docker-exec() {
  docker-compose -f docker-compose.yml exec -e VAULT_TOKEN=${VAULT_TOKEN:-root} -T $1 sh -c "${@:2}"
}

function unseal-vault() {
  ROOT_VAULT_OPERATOR_OUTPUT=$(cat /tmp/vault_root_operator_output)
  ROOT_VAULT_TOKEN=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Initial Root Token:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_1=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 1:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_2=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 2:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_3=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 3:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_4=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 4:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_5=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 5:" | sed 's/.*: *//')

  AUTH_VAULT_OPERATOR_OUTPUT=$(cat /tmp/vault_auth_operator_output)
  AUTH_VAULT_TOKEN=$(echo "$AUTH_VAULT_OPERATOR_OUTPUT" | grep "Initial Root Token:" | sed 's/.*: *//')
  AUTH_VAULT_UNSEAL_KEY_1=$(echo "$AUTH_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 1:" | sed 's/.*: *//')
  AUTH_VAULT_UNSEAL_KEY_2=$(echo "$AUTH_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 2:" | sed 's/.*: *//')
  AUTH_VAULT_UNSEAL_KEY_3=$(echo "$AUTH_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 3:" | sed 's/.*: *//')
  AUTH_VAULT_UNSEAL_KEY_4=$(echo "$AUTH_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 4:" | sed 's/.*: *//')
  AUTH_VAULT_UNSEAL_KEY_5=$(echo "$AUTH_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 5:" | sed 's/.*: *//')

  docker-exec "vault_root" "vault operator unseal $ROOT_VAULT_UNSEAL_KEY_1" > /dev/null 2>&1
  docker-exec "vault_root" "vault operator unseal $ROOT_VAULT_UNSEAL_KEY_2" > /dev/null 2>&1
  docker-exec "vault_root" "vault operator unseal $ROOT_VAULT_UNSEAL_KEY_3" > /dev/null 2>&1

  docker-exec "vault_auth" "vault operator unseal $AUTH_VAULT_UNSEAL_KEY_1" > /dev/null 2>&1
  docker-exec "vault_auth" "vault operator unseal $AUTH_VAULT_UNSEAL_KEY_2" > /dev/null 2>&1
  docker-exec "vault_auth" "vault operator unseal $AUTH_VAULT_UNSEAL_KEY_3" > /dev/null 2>&1
}

unseal-vault
