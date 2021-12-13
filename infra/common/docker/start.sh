#!/usr/bin/env bash

set -e

export VAULT_AUTH="https://ew3a1.auth.flant-sandbox.flant.com/"
export VAULT_ROOT_SOURCE="https://root-source.flant-sandbox.flant.com"

if [ -z "$VAULT_AUTH_TOKEN" ]; then
  echo VAULT_AUTH_TOKEN is unset
  exit 1
fi

if [ -z "$VAULT_ROOT_SOURCE_TOKEN" ]; then
  echo VAULT_ROOT_SOURCE_TOKEN is unset
  exit 1
fi

echo enable plugins
#VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault auth enable -path=flant_iam_auth flant_iam_auth
#VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault secrets enable -path=ssh ssh
#VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault secrets enable -path=flant_iam flant_iam
#VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault auth enable -path=flant_iam_auth flant_iam_auth

echo prepare flant_iam on root source vault
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write -force auth/flant_iam_auth/kafka/generate_csr > /dev/null 2>&1
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=negentropy-kafka-1.negentropy.flant.local:9093,negentropy-kafka-2.negentropy.flant.local:9093,negentropy-kafka-3.negentropy.flant.local:9093
root_pubkey=$(VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault read flant_iam/kafka/public_key | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')
echo DEBUG: root_pubkey is $root_pubkey

echo prepare flant_iam_auth on root source vault
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write -force flant_iam/kafka/generate_csr > /dev/null 2>&1
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write flant_iam/kafka/configure_access kafka_endpoints=negentropy-kafka-1.negentropy.flant.local:9093,negentropy-kafka-2.negentropy.flant.local:9093,negentropy-kafka-3.negentropy.flant.local:9093
root_auth_pubkey=$(VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault read auth/flant_iam_auth/kafka/public_key | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')
echo DEBUG: root_auth_pubkey is $root_auth_pubkey

echo prepare flant_iam_auth on auth vault
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write -force auth/flant_iam_auth/kafka/generate_csr > /dev/null 2>&1
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=negentropy-kafka-1.negentropy.flant.local:9093,negentropy-kafka-2.negentropy.flant.local:9093,negentropy-kafka-3.negentropy.flant.local:9093
auth_auth_pubkey=$(VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault read auth/flant_iam_auth/kafka/public_key | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')
echo auth_auth_pubkey is $auth_auth_pubkey

echo configure flant_iam on root source vault
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write flant_iam/kafka/configure self_topic_name=root_source peers_public_keys="$root_auth_pubkey","$auth_auth_pubkey"

echo create replica for root source vault flant_iam_auth
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write flant_iam/replica/auth-1 type=Vault public_key="$root_auth_pubkey"

echo create replica for auth vault flant_iam_auth
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write flant_iam/replica/auth-2 type=Vault public_key="$auth_auth_pubkey"

echo configure flant_iam_auth on root source vault
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write auth/flant_iam_auth/kafka/configure peers_public_keys="$root_pubkey" self_topic_name=auth-source.auth-1 root_topic_name=root_source.auth-1 root_public_key="$root_pubkey"

echo configure flant_iam_auth on auth vault
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write auth/flant_iam_auth/kafka/configure peers_public_keys="$root_pubkey" self_topic_name=auth-source.auth-2 root_topic_name=root_source.auth-2 root_public_key="$root_pubkey"

echo configure server_access on root source vault
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write flant_iam/configure_extension/server_access roles_for_servers=servers role_for_ssh_access=ssh name=ssh delete_expired_password_seeds_after=1000000 expire_password_seed_after_reveal_in=1000000 last_allocated_uid=10000 --format=json
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json

echo configure server_access on auth vault
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json

echo enable jwt
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write -force flant_iam/jwt/enable > /dev/null 2>&1
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write -force auth/flant_iam_auth/jwt/enable > /dev/null 2>&1
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write -force auth/flant_iam_auth/jwt/enable > /dev/null 2>&1

echo create policy
cat <<'EOF' > /tmp/full.hcl
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF

echo enable approle
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault auth enable approle
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault auth enable approle

echo load policy
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault policy write full /tmp/full.hcl
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault policy write full /tmp/full.hcl

echo configure approle
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write auth/approle/role/full secret_id_ttl=15m token_ttl=1000s token_policies=full
root_secretID=$(VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write -format=json -f auth/approle/role/full/secret-id | jq -r '.data.secret_id')
echo DEBUG: root_secretID is $root_secretID
root_roleID=$(VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault read -format=json auth/approle/role/full/role-id | jq -r '.data.role_id')
echo DEBUG: root_roleID is $root_roleID
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write auth/approle/role/full secret_id_ttl=15m token_ttl=1000s token_policies=full
auth_secretID=$(VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write -format=json -f auth/approle/role/full/secret-id | jq -r '.data.secret_id')
echo DEBUG: auth_secretID is $auth_secretID
auth_roleID=$(VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault read -format=json auth/approle/role/full/role-id | jq -r '.data.role_id')
echo DEBUG: auth_roleID is $auth_roleID

echo configure self-access
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write auth/flant_iam_auth/configure_vault_access \
  vault_addr="https://10.20.1.11:443" \
  vault_tls_server_name="vault_host" \
  role_name="full" \
  secret_id_ttl="15m" \
  approle_mount_point="/auth/approle/" \
  secret_id="$root_secretID" \
  role_id="$root_roleID" \
  vault_api_ca=""
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write auth/flant_iam_auth/configure_vault_access \
  vault_addr="http://127.0.0.1:8200" \
  vault_tls_server_name="vault_host" \
  role_name="full" \
  secret_id_ttl="15m" \
  approle_mount_point="/auth/approle/" \
  secret_id="$auth_secretID" \
  role_id="$auth_roleID" \
  vault_api_ca=""

echo configure multipass
VAULT_ADDR=$VAULT_ROOT_SOURCE VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN vault write auth/flant_iam_auth/auth_method/multipass \
  token_ttl="30m" \
  token_policies="full" \
  token_no_default_policy=true \
  method_type="multipass_jwt"
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write auth/flant_iam_auth/auth_method/multipass \
  token_ttl="30m" \
  token_policies="full" \
  token_no_default_policy=true \
  method_type="multipass_jwt"

echo configure ssh plugin on auth vault only
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write ssh/config/ca \
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
VAULT_ADDR=$VAULT_AUTH VAULT_TOKEN=$VAULT_AUTH_TOKEN vault write ssh/roles/signer - <<"EOF"
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
EOF
