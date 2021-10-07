#!/usr/bin/env bash

function docker-exec() {
  docker-compose -f docker-compose.yml exec -e VAULT_TOKEN=${VAULT_TOKEN:-root} -T $1 sh -c "${@:2}"
}

function init-vault() {
  docker-exec "vault_root" "vault operator init" > /tmp/vault_root_operator_output
  ROOT_VAULT_OPERATOR_OUTPUT=$(cat /tmp/vault_root_operator_output)
  ROOT_VAULT_TOKEN=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Initial Root Token:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_1=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 1:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_2=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 2:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_3=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 3:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_4=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 4:" | sed 's/.*: *//')
  ROOT_VAULT_UNSEAL_KEY_5=$(echo "$ROOT_VAULT_OPERATOR_OUTPUT" | grep "Unseal Key 5:" | sed 's/.*: *//')

  docker-exec "vault_auth" "vault operator init" > /tmp/vault_auth_operator_output
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

  echo ROOT_VAULT_TOKEN is $ROOT_VAULT_TOKEN
  echo AUTH_VAULT_TOKEN is $AUTH_VAULT_TOKEN
}

function check-vault() {
  vault="vault_$1"
  docker-exec "$vault" "until vault status &> /dev/null; do sleep 1; done"
}

function activate-plugin() {
  plugin="$1"

  # flant_iam_auth is auth plugin, so for it need to use different enable command
  if [ $plugin == "flant_iam_auth" ]; then
      VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault auth enable -path=$plugin $plugin"
      VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault auth enable -path=$plugin $plugin"
  # flant_iam only needs on root source vault
  elif [ $plugin == "flant_iam" ]; then
      VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault secrets enable -path=$plugin $plugin"
  else
      VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault secrets enable -path=$plugin $plugin"
      VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault secrets enable -path=$plugin $plugin"
  fi
}

function connect_plugins() {
  # prepare flant_iam on root source vault
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write -force flant_iam/kafka/generate_csr" > /dev/null 2>&1
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write flant_iam/kafka/configure_access kafka_endpoints=kafka:9092"
  VAULT_TOKEN=$ROOT_VAULT_TOKEN root_pubkey=$(docker-exec "vault_root" "vault read flant_iam/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # prepare flant_iam_auth on root source vault
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write -force auth/flant_iam_auth/kafka/generate_csr" > /dev/null 2>&1
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
  VAULT_TOKEN=$ROOT_VAULT_TOKEN root_auth_pubkey=$(docker-exec "vault_root" "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # prepare flant_iam_auth on auth vault
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault write -force auth/flant_iam_auth/kafka/generate_csr" > /dev/null 2>&1
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
  VAULT_TOKEN=$AUTH_VAULT_TOKEN auth_auth_pubkey=$(docker-exec "vault_auth" "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # configure flant_iam on root source vault
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write flant_iam/kafka/configure self_topic_name=root_source peers_public_keys=\"$root_auth_pubkey\",\"$auth_auth_pubkey\""

  # configure flant_iam_auth on root source vault
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" \
    "vault write auth/flant_iam_auth/kafka/configure peers_public_keys=\"$root_pubkey\" self_topic_name=auth-source.auth-1 root_topic_name=root_source.auth-1 root_public_key=\"$root_pubkey\""

  # configure flant_iam_auth on auth vault
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" \
    "vault write auth/flant_iam_auth/kafka/configure peers_public_keys=\"$root_pubkey\" self_topic_name=auth-source.auth-2 root_topic_name=root_source.auth-2 root_public_key=\"$root_pubkey\""

  # create replica for root source vault flant_iam_auth
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write flant_iam/replica/auth-1 type=Vault public_key=\"$root_auth_pubkey\""

  # create replica for auth vault flant_iam_auth
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write flant_iam/replica/auth-2 type=Vault public_key=\"$auth_auth_pubkey\""
}

function initialize() {
  # configure server_access on root source vault
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" \
    "vault write flant_iam/configure_extension/server_access roles_for_servers=servers role_for_ssh_access=ssh name=ssh delete_expired_password_seeds_after=1000000 expire_password_seed_after_reveal_in=1000000 last_allocated_uid=10000 --format=json"
  # TODO: why we configure this in root source vault?
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  # configure server_access on auth vault
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  # enable jwt
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write -force flant_iam/jwt/enable" > /dev/null 2>&1
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write -force auth/flant_iam_auth/jwt/enable" > /dev/null 2>&1
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault write -force auth/flant_iam_auth/jwt/enable" > /dev/null 2>&1

  # create policy
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec 'vault_root' 'cat <<'EOF' > full.hcl
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF'
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec 'vault_auth' 'cat <<'EOF' > full.hcl
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF'

  # enable approle
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault auth enable approle"
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault auth enable approle"

  # load policy
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault policy write full full.hcl"
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault policy write full full.hcl"

  # configure approle
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write auth/approle/role/full secret_id_ttl=30m token_ttl=900s token_policies=full"
  VAULT_TOKEN=$ROOT_VAULT_TOKEN root_secretID=$(docker-exec "vault_root" "vault write -format=json -f auth/approle/role/full/secret-id" | jq -r '.data.secret_id')
  VAULT_TOKEN=$ROOT_VAULT_TOKEN root_roleID=$(docker-exec "vault_root" "vault read -format=json auth/approle/role/full/role-id" | jq -r '.data.role_id')
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault write auth/approle/role/full secret_id_ttl=30m token_ttl=900s token_policies=full"
  VAULT_TOKEN=$AUTH_VAULT_TOKEN auth_secretID=$(docker-exec "vault_auth" "vault write -format=json -f auth/approle/role/full/secret-id" | jq -r '.data.secret_id')
  VAULT_TOKEN=$AUTH_VAULT_TOKEN auth_roleID=$(docker-exec "vault_auth" "vault read -format=json auth/approle/role/full/role-id" | jq -r '.data.role_id')

  # renew root token
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault token create -orphan -policy=root -field=token" > /tmp/root_token
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault token create -orphan -policy=root -field=token" > /tmp/auth_token
  export ROOT_VAULT_TOKEN="$(cat /tmp/root_token)"
  export AUTH_VAULT_TOKEN="$(cat /tmp/auth_token)"

  # configure self-access
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write auth/flant_iam_auth/configure_vault_access \
    vault_addr=\"http://127.0.0.1:8200\" \
    vault_tls_server_name=\"vault_host\" \
    role_name=\"full\" \
    secret_id_ttl=\"30m\" \
    approle_mount_point=\"/auth/approle/\" \
    secret_id=\"$root_secretID\" \
    role_id=\"$root_roleID\" \
    vault_api_ca=\"\""
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault write auth/flant_iam_auth/configure_vault_access \
    vault_addr=\"http://127.0.0.1:8200\" \
    vault_tls_server_name=\"vault_host\" \
    role_name=\"full\" \
    secret_id_ttl=\"30m\" \
    approle_mount_point=\"/auth/approle/\" \
    secret_id=\"$auth_secretID\" \
    role_id=\"$auth_roleID\" \
    vault_api_ca=\"\""

  # configure multipass
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault_root" "vault write auth/flant_iam_auth/auth_method/multipass \
    token_ttl=\"30m\" \
    token_policies=\"full\" \
    token_no_default_policy=true \
    method_type=\"multipass_jwt\""
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault_auth" "vault write auth/flant_iam_auth/auth_method/multipass \
    token_ttl=\"30m\" \
    token_policies=\"full\" \
    token_no_default_policy=true \
    method_type=\"multipass_jwt\""

  # configure ssh plugin on auth vault only
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec 'vault_auth' 'vault write ssh/config/ca \
	  private_key="-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA0/G1wVnF9ufvio1W1XBAD51EU6UP+p0otMVfpap/7DgkyZY0
WEzJNFGxmR271VdnnWGKYApAyjlhfXheYaY5j2rMmKLJFTCc/X2ntfnJfqZsnJxk
2S7KYNK+fTa/++68o2tipJZWOAl3O85Zrv0ft9elYM6Vj8keNNO5SGZdvAQGoW3X
yif4zaaZFWS+Nd60hWeYEwZTCFZmataVLzgbWoTKx9ig71nYNFCVoeao8h8Ynwvi
797x1pSqsC64CRUPOfVeLG306obeNV8LfNJ5CkgO8ji+BZ8RcMSauQ0iW+chk2J7
b902JcJpWZi9yYNeEt2kM1vNCG1bkcJw38L9JQIDAQABAoIBABSABaeNCmPmbToG
j8aXU+ruuEQq7A++kchiauz4P+VWTOCewbNkwfVojXgU8y0ghion3B2MAFZPFInx
UZe6X0jq+J0u6ao+CIFQXR9x6LZyXIENc4e6SeLxn3E3EXzJy782zNTEodRLvhev
zubpHt9GYX2qnbbJqj1L2VkSZbCgufku+4y4UbFINMImzwU9kZpc3rbqsYCSzNH+
x7cCsj1yuXK4Du+k5NX16jFnuZfES05h6Rq26egSkBSrhzTd8eP6YVun6JnJEVOw
vOqGyGVFMu5toOb8Wnjp5PEj6/c4oRzg+t1tXr1YUoo3RAA17JnqeHopVb8gz1d+
83bxpEUCgYEA8PiWUZ8Za++w1iE1XPt499504pwSzPTh5vbTl0nbE17YSfA0Dc4S
vyrZZLjmYKezqebM1Sw9/IJWblk4e6UbcRu0+XsQpeH2Yxv0h/fJTi43tYVyzSKP
70+IYJJBFJ3xfA8dPN8HqvkKUMHcQvdwU2DEC47wg15yrD0+sETF83cCgYEA4Smr
603VY5HB/Ic+ehAXMc/CFRB6bs2ytxJL254bmPWJablqHH25xYbe7weJEPGJedaw
Ek1r3hFjGddxLC4ix5i6YfH4NwRMBh0rU8YmAWHVyHVFlZecGTv+42dBxXzVxPS9
Hf/DFLy6r3L0FL+pcVxRy9Mm63e3ydnF54ptI0MCgYAHQDOluRfWu5uildU5Owfk
zXjO6MtYB3ZUsNClGL/S0WPItcWbNLwzrGJmOXoVJnatghhfwbkLxBA9ucmNTuaI
fMDxUNarZyU2zjyJatdP1uwuNhnCOmwCU25TGZODv0zo4ruKfVuJtXyt+WdbTH7A
w4SipGZwTYM904nzW95o+QKBgHRWmbO8xZLqzvZx0sAy7CkalcdYekoiEkMxOuzA
prXDuDpeSQtrkr8SzsFmfVW51zSSzurGAgP9q9zASoNvWx0SNstAwOV8XOOT0r04
Vo7ERDeNEGUYrtkC/NH2mi82LyXS5pxHeD6QvUzF8oN9/EjMUJ8l/KgRdW7gDLdz
+KwNAoGAQkNO/RWEsJYUkEUkkObfSqGN75s78fjT1yZ7CX0dUvHv6KC3+f7RmNHM
2zNxHZ+s+x9hfasJaduoV/hksluY4KUMuZjkfih8CaRIqCY8E/wEYjsyYJzJ4f1u
C+iz1LopgyIrKSebDzl13Yx9/J6dP3LrC+TiYyYl0bf4a4AStLw=
-----END RSA PRIVATE KEY-----" \
          public_key="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDT8bXBWcX25++KjVbVcEAPnURTpQ/6nSi0xV+lqn/sOCTJljRYTMk0UbGZHbvVV2edYYpgCkDKOWF9eF5hpjmPasyYoskVMJz9fae1+cl+pmycnGTZLspg0r59Nr/77ryja2KkllY4CXc7zlmu/R+316VgzpWPyR4007lIZl28BAahbdfKJ/jNppkVZL413rSFZ5gTBlMIVmZq1pUvOBtahMrH2KDvWdg0UJWh5qjyHxifC+Lv3vHWlKqwLrgJFQ859V4sbfTqht41Xwt80nkKSA7yOL4FnxFwxJq5DSJb5yGTYntv3TYlwmlZmL3Jg14S3aQzW80IbVuRwnDfwv0l"
'

  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec 'vault_auth' 'vault write ssh/roles/signer - <<"EOF"
{
  "allow_user_certificates": true,
  "algorithm_signer": "rsa-sha2-256",
  "allowed_users": "*",
  "allowed_extensions": "permit-pty,permit-agent-forwarding",
  "default_extensions": [
    {
      "permit-pty": "",
      "permit-agent-forwarding": ""
    }
  ],
  "key_type": "ca",
  "ttl": "5m0s"
}
EOF"' > /dev/null 2>&1
}

docker run --rm -v $(pwd):/app -w /app/infra/common/vault golang:1.16.8-alpine sh -c "apk add bash git make musl-dev gcc patch && ./build_vault.sh"

docker-compose -f docker-compose.yml up -d minio

while true; do
  docker run --rm --network=negentropy_default --entrypoint=sh minio/mc -c "mc config host add minio http://minio:9000 minio minio123 && mc mb minio/vault-root && mc mb minio/vault-auth"
  status=$?
  if [ $status -eq 0 ]; then
    break
  fi
done

docker-compose -f docker-compose.yml up -d

echo "DEBUG: sleep for 5s"
sleep 5

init-vault

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
