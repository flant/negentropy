#!/usr/bin/env bash

function docker-exec() {
  docker-compose -f docker-compose2.yml exec -e VAULT_TOKEN=${VAULT_TOKEN:-root} -T $1 sh -c "${@:2}"
}

function check-vault() {
  vault="$1"
  docker-exec "$vault" "until vault status &> /dev/null; do sleep 1; done"
}

function activate-plugin() {
  plugin="$1"

  # flant_iam_auth is auth plugin, so for it need to use different enable command
  if [ $plugin == "flant_iam_auth" ]; then
      docker-exec "vault_auth" "vault auth enable -path=$plugin $plugin"
      docker-exec "vault_root" "vault auth enable -path=$plugin $plugin"
  # flant_iam only needs on root source vault
  elif [ $plugin == "flant_iam" ]; then
      docker-exec "vault_root" "vault secrets enable -path=$plugin $plugin"
  else
      docker-exec "vault_auth" "vault secrets enable -path=$plugin $plugin"
      docker-exec "vault_root" "vault secrets enable -path=$plugin $plugin"
  fi
}

function connect_plugins() {
  # prepare flant_iam on root source vault
  docker-exec "vault_root" "vault write -force flant_iam/kafka/generate_csr" > /dev/null 2>&1
  docker-exec "vault_root" "vault write flant_iam/kafka/configure_access kafka_endpoints=kafka:9092"
  root_pubkey=$(docker-exec "vault_root" "vault read flant_iam/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # prepare flant_iam_auth on root source vault
  docker-exec "vault_root" "vault write -force auth/flant_iam_auth/kafka/generate_csr" > /dev/null 2>&1
  docker-exec "vault_root" "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
  root_auth_pubkey=$(docker-exec "vault_root" "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # prepare flant_iam_auth on auth vault
  docker-exec "vault_auth" "vault write -force auth/flant_iam_auth/kafka/generate_csr" > /dev/null 2>&1
  docker-exec "vault_auth" "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
  auth_auth_pubkey=$(docker-exec "vault_auth" "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # configure flant_iam on root source vault
  docker-exec "vault_root" "vault write flant_iam/kafka/configure self_topic_name=root_source peers_public_keys=\"$root_auth_pubkey\",\"$auth_auth_pubkey\""

  # configure flant_iam_auth on root source vault
  docker-exec "vault_root" \
    "vault write auth/flant_iam_auth/kafka/configure peers_public_keys=\"$root_pubkey\" self_topic_name=auth-source.auth-1 root_topic_name=root_source.auth-1 root_public_key=\"$root_pubkey\""

  # configure flant_iam_auth on auth vault
  docker-exec "vault_auth" \
    "vault write auth/flant_iam_auth/kafka/configure peers_public_keys=\"$root_pubkey\" self_topic_name=auth-source.auth-2 root_topic_name=root_source.auth-2 root_public_key=\"$root_pubkey\""

  # create replica for root source vault flant_iam_auth
  docker-exec "vault_root" "vault write flant_iam/replica/auth-1 type=Vault public_key=\"$root_auth_pubkey\""

  # create replica for auth vault flant_iam_auth
  docker-exec "vault_root" "vault write flant_iam/replica/auth-2 type=Vault public_key=\"$auth_auth_pubkey\""
}

function initialize() {
  # configure server_access on root source vault
  docker-exec "vault_root" \
    "vault write flant_iam/configure_extension/server_access roles_for_servers=servers role_for_ssh_access=ssh name=ssh delete_expired_password_seeds_after=1000000 expire_password_seed_after_reveal_in=1000000 last_allocated_uid=10000 --format=json"
  # TODO: why we configure this in root source vault?
  docker-exec "vault_root" "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  # configure server_access on auth vault
  docker-exec "vault_auth" "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  # enable jwt
  docker-exec "vault_root" "vault write -force flant_iam/jwt/enable" > /dev/null 2>&1
  docker-exec "vault_root" "vault write -force auth/flant_iam_auth/jwt/enable" > /dev/null 2>&1
  docker-exec "vault_auth" "vault write -force auth/flant_iam_auth/jwt/enable" > /dev/null 2>&1
}

docker run -it --rm -v $(pwd):/app -w /app/infra/common/vault golang:1.16.8-alpine sh -c "apk add bash git make musl-dev gcc patch && ./build_vault.sh"

docker-compose -f docker-compose2.yml up -d

vaults=(auth root)
for v in "${vaults[@]}"
do
  check-vault "$v"
done

plugins=(flant_iam flant_iam_auth ssh)
for i in "${plugins[@]}"
do
  activate-plugin "$i"
done

connect_plugins

initialize
