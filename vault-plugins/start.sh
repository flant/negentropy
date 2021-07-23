#!/usr/bin/env bash

set -e

function docker-exec() {
  docker-compose exec -e VAULT_TOKEN=${VAULT_TOKEN:-root} -T vault sh -c "$@"
}

function check-vault() {
  docker-exec "until vault status &> /dev/null; do sleep 1; done"
}

function activate-plugin() {
  plugin="$1"

  # flant_iam_auth is auth plugin, so for it need to use different enable command
  if [ $plugin == "flant_iam_auth" ]; then
      docker-exec "vault auth enable -path=$plugin $plugin"
  else
      docker-exec "vault secrets enable -path=$plugin $plugin"
  fi
}

function connect_plugins() {
  # prepare flant_iam
  docker-exec "vault write -force flant_iam/kafka/generate_csr" >/dev/null 2>&1
  docker-exec "vault write flant_iam/kafka/configure_access kafka_endpoints=kafka:9092"
  root_pubkey=$(docker-exec "vault read flant_iam/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # prepare flant_iam_auth
  docker-exec "vault write -force auth/flant_iam_auth/kafka/generate_csr" >/dev/null 2>&1
  docker-exec "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
  auth_pubkey=$(docker-exec "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # configure flant_iam
  docker-exec "vault write flant_iam/kafka/configure self_topic_name=root_source"

  # configure flant_iam_auth
  docker-exec \
    "vault write auth/flant_iam_auth/kafka/configure self_topic_name=auth-source.auth-1 root_topic_name=root_source.auth-1 root_public_key=\"$root_pubkey\""

  # create replica
  docker-exec "vault write flant_iam/replica/auth-1 type=Vault public_key=\"$auth_pubkey\""
}

function initialize() {
  docker-exec "vault write flant_iam/configure_extension/server_access roles_for_servers=servers role_for_ssh_access=ssh name=ssh delete_expired_password_seeds_after=1000000 expire_password_seed_after_reveal_in=1000000 last_allocated_uid=10000 --format=json"
  docker-exec "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  docker-exec "vault write -force flant_iam/jwt/enable" >/dev/null 2>&1
  docker-exec "vault write -force auth/flant_iam_auth/jwt/enable" >/dev/null 2>&1

  # TODO: why?
  docker-exec "vault token create -orphan -policy=root -field=token" > /tmp/root_token
  export VAULT_TOKEN="$(cat /tmp/root_token)"

  docker-exec "vault token create -orphan -policy=root -field=token > /vault/testdata/token"

  docker-exec 'cat <<'EOF' > full.hcl
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF'

  docker-exec "vault auth enable approle"
  docker-exec "vault policy write full full.hcl"
  docker-exec "vault write auth/approle/role/full secret_id_ttl=30m token_ttl=900s token_policies=full"
  secretID=$(docker-exec "vault write -format=json -f auth/approle/role/full/secret-id" | jq -r '.data.secret_id')
  roleID=$(docker-exec "vault read -format=json auth/approle/role/full/role-id" | jq -r '.data.role_id')

  docker-exec "vault write auth/flant_iam_auth/configure_vault_access \
    vault_addr=\"http://127.0.0.1:8200\" \
    vault_tls_server_name=\"vault_host\" \
    role_name=\"full\" \
    secret_id_ttl=\"120m\" \
    approle_mount_point=\"/auth/approle/\" \
    secret_id=\"$secretID\" \
    role_id=\"$roleID\" \
    vault_api_ca=\"\""

  docker-exec "vault write auth/flant_iam_auth/auth_method/multipass \
    token_ttl=\"30m\" \
    token_policies=\"full\" \
    token_no_default_policy=true \
    method_type=\"multipass_jwt\""
}

docker-compose up -d

check-vault

plugins=(flant_iam flant_iam_auth ssh)
for i in "${plugins[@]}"
do
  activate-plugin "$i"
done

connect_plugins

initialize
