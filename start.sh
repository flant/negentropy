#!/usr/bin/env bash

set -e


while [[ $# -gt 0 ]]; do
  case $1 in
    -d|--debug)
    export DEBUG="true"
    ;;
    *)
    echo "Unknown parameter $1"; exit 1
    ;;
  esac
  shift
done

export COMPOSE_PROJECT_NAME=negentropy

function docker-exec() {
  if [[ $DEBUG == "true" ]]; then
    docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.debug.yml exec -e VAULT_TOKEN=${VAULT_TOKEN:-root} -T $1 sh -c "${@:2}"
  else
    docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml exec -e VAULT_TOKEN=${VAULT_TOKEN:-root} -T $1 sh -c "${@:2}"
  fi
}

function initialize() {
  # configure server_access on root source vault
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" \
    "vault write flant_iam/configure_extension/server_access roles_for_servers=servers role_for_ssh_access=ssh name=ssh delete_expired_password_seeds_after=1000000 expire_password_seed_after_reveal_in=1000000 last_allocated_uid=10000 --format=json"
  # TODO: why we configure this in root source vault?
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  # configure server_access on auth vault
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

  # enable jwt
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault write -force flant_iam/jwt/enable" > /dev/null 2>&1
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault write -force auth/flant_iam_auth/jwt/enable" > /dev/null 2>&1
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault write -force auth/flant_iam_auth/jwt/enable" > /dev/null 2>&1

  # create policy
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec 'vault-root' 'cat <<'EOF' > full.hcl
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF'
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec 'vault-auth' 'cat <<'EOF' > full.hcl
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF'

  # enable approle
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault auth enable approle"
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault auth enable approle"

  # load policy
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault policy write full full.hcl"
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault policy write full full.hcl"

  # configure approle
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault write auth/approle/role/full secret_id_ttl=5m token_ttl=120s token_policies=full"
  VAULT_TOKEN=$ROOT_VAULT_TOKEN root_secretID=$(docker-exec "vault-root" "vault write -format=json -f auth/approle/role/full/secret-id" | jq -r '.data.secret_id')
  VAULT_TOKEN=$ROOT_VAULT_TOKEN root_roleID=$(docker-exec "vault-root" "vault read -format=json auth/approle/role/full/role-id" | jq -r '.data.role_id')
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault write auth/approle/role/full secret_id_ttl=5m token_ttl=120s token_policies=full"
  VAULT_TOKEN=$AUTH_VAULT_TOKEN auth_secretID=$(docker-exec "vault-auth" "vault write -format=json -f auth/approle/role/full/secret-id" | jq -r '.data.secret_id')
  VAULT_TOKEN=$AUTH_VAULT_TOKEN auth_roleID=$(docker-exec "vault-auth" "vault read -format=json auth/approle/role/full/role-id" | jq -r '.data.role_id')

  # renew root token
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault token create -orphan -policy=root -field=token" > /tmp/root_token
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault token create -orphan -policy=root -field=token" > /tmp/auth_token
  export ROOT_VAULT_TOKEN="$(cat /tmp/root_token)"
  export AUTH_VAULT_TOKEN="$(cat /tmp/auth_token)"

  # configure self-access
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault write auth/flant_iam_auth/configure_vault_access \
    vault_addr=\"http://127.0.0.1:8200\" \
    vault_tls_server_name=\"vault_host\" \
    role_name=\"full\" \
    secret_id_ttl=\"5m\" \
    approle_mount_point=\"/auth/approle/\" \
    secret_id=\"$root_secretID\" \
    role_id=\"$root_roleID\" \
    vault_api_ca=\"\""
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault write auth/flant_iam_auth/configure_vault_access \
    vault_addr=\"http://127.0.0.1:8200\" \
    vault_tls_server_name=\"vault_host\" \
    role_name=\"full\" \
    secret_id_ttl=\"5m\" \
    approle_mount_point=\"/auth/approle/\" \
    secret_id=\"$auth_secretID\" \
    role_id=\"$auth_roleID\" \
    vault_api_ca=\"\""

  # configure multipass
  VAULT_TOKEN=$ROOT_VAULT_TOKEN docker-exec "vault-root" "vault write auth/flant_iam_auth/auth_method/multipass \
    token_ttl=\"30m\" \
    token_policies=\"full\" \
    token_no_default_policy=true \
    method_type=\"multipass_jwt\""
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec "vault-auth" "vault write auth/flant_iam_auth/auth_method/multipass \
    token_ttl=\"30m\" \
    token_policies=\"full\" \
    token_no_default_policy=true \
    method_type=\"multipass_jwt\""

  # configure ssh plugin on auth vault only
  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec 'vault-auth' 'vault write ssh/config/ca \
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

  VAULT_TOKEN=$AUTH_VAULT_TOKEN docker-exec 'vault-auth' 'vault write ssh/roles/signer - <<"EOF"
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

if [[ $DEBUG == "true" ]]; then
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.debug.yml up -d
  echo "DEBUG: run you debug tools and connect to dlv servers at both vaults"
  while true; do
    read -p "Are you ready? (Y/n) " ANSWER;
      if [[ -z "$ANSWER" ]]; then ANSWER=Y; fi
      case $ANSWER in
          [Yy]* ) break;;
          [Nn]* ) exit 1;;
      esac
  done
else
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml up -d
  echo "DEBUG: sleep for 5s while containers starts"
  sleep 5
fi

pip install virtualenv
virtualenv scripts/e2e
source scripts/e2e/bin/activate
 pip install -r scripts/requirements.txt
python scripts/start.py

deactivate

export ROOT_VAULT_TOKEN=$(cat /tmp/root_token)
export AUTH_VAULT_TOKEN=$(cat /tmp/auth_token)

echo $ROOT_VAULT_TOKEN
echo $AUTH_VAULT_TOKEN

initialize

