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
  docker-exec "vault write -force flant_iam/kafka/generate_csr" > /dev/null 2>&1
  docker-exec "vault write flant_iam/kafka/configure_access kafka_endpoints=kafka:9092"
  root_pubkey=$(docker-exec "vault read flant_iam/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # prepare flant_iam_auth
  docker-exec "vault write -force auth/flant_iam_auth/kafka/generate_csr" > /dev/null 2>&1
  docker-exec "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
  auth_pubkey=$(docker-exec "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # configure flant_iam
  docker-exec "vault write flant_iam/kafka/configure self_topic_name=root_source peers_public_keys=\"$auth_pubkey\""

  # configure flant_iam_auth
  docker-exec \
    "vault write auth/flant_iam_auth/kafka/configure peers_public_keys=\"$root_pubkey\" self_topic_name=auth-source.auth-1 root_topic_name=root_source.auth-1 root_public_key=\"$root_pubkey\""

  # create replica
  docker-exec "vault write flant_iam/replica/auth-1 type=Vault public_key=\"$auth_pubkey\""
}

function initialize() {
  docker-exec "vault write flant_iam/configure_extension/server_access roles_for_servers=servers role_for_ssh_access=ssh name=ssh delete_expired_password_seeds_after=1000000 expire_password_seed_after_reveal_in=1000000 last_allocated_uid=10000 --format=json"
  docker-exec "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  docker-exec "vault write -force flant_iam/jwt/enable" > /dev/null 2>&1
  docker-exec "vault write -force auth/flant_iam_auth/jwt/enable" > /dev/null 2>&1

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

  docker-exec 'vault write ssh/config/ca \
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

  docker-exec 'vault write ssh/roles/signer - <<"EOF"
{
  "allow_user_certificates": true,
  "allowed_users": "*",
  "allowed_extensions": "permit-pty,permit-agent-forwarding",
  "default_extensions": [
    {
      "permit-pty": "",
      "permit-agent-forwarding": ""
    }
  ],
  "key_type": "ca",
  "ttl": "2m0s"
}
EOF"' > /dev/null 2>&1
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
