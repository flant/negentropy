#!/bin/bash

# It's a simple script to obtain a JWT

cat <<'EOF' | docker exec -i "dev-vault" ash -

set -eo pipefail

which jq || apk add jq

ROOT_TOKEN=root

echo Configure "identity oidc"

vault write identity/oidc/config issuer=http://127.0.0.1:8200
vault write identity/oidc/key/mykey allowed_client_ids=* algorithm=EdDSA
vault write identity/oidc/role/myrole key=mykey client_id=authd template='{"username":{{identity.entity.name}} }'

echo Create policy

cat <<'POLICY' | vault policy write testuserpolicy -
path "identity/oidc/token/myrole" {
  capabilities = ["read"]
}

path "ssh/creds/otp_key_role" {
  capabilities = ["update"]
}

POLICY

echo Create entity and enitity_alias
vault write /identity/entity name=testuser policies=testuserpolicy
ENTITY_ID=$(vault read -format=json /identity/entity/name/testuser | jq -r '.data.id')

MOUNT_ACCESSOR=$(vault auth list -format=json | jq -r '."token/".accessor')

vault write identity/entity-alias name=testuser-alias canonical_id=$ENTITY_ID mount_accessor=$MOUNT_ACCESSOR

# List all enitity aliases
# for id in $(vault list -format=json /identity/entity-alias/id | jq '.[]' -r ) ; do vault read -format=json /identity/entity-alias/id/$id | jq '.data.id + " " + .data.name + " " + (.data.metadata|tostring)' ; done

# List all enitities
# for id in $(vault list -format=json /identity/entity/id | jq '.[]' -r ) ; do vault read -format=json /identity/entity/id/$id | jq '.data.id + " " + .data.name + " " + (.data.metadata|tostring)' ; done

echo Create role for auth token

vault write auth/token/roles/myrole allowed_entity_aliases=* allowed_policies=testuserpolicy

echo Create auth token for entity alias

ENTITY_TOKEN=$(vault write -format=json auth/token/create/myrole \
   entity_alias="testuser-alias" |  jq -r '.auth.client_token')

echo Mount jwt auth plugin at myjwt path
vault auth enable jwt -path myjwt # /auth/myjwt/...

echo Configure auth/jwt to trust itself and use tokens from identity/oidc
vault write auth/myjwt/config jwks_url=http://127.0.0.1:8200/v1/identity/oidc/.well-known/keys bound_issuer=https://127.0.0.1:8200/v1/identity/oidc

echo Create authd role in myjwt
vault write /auth/myjwt/role/authd role_type=jwt bound_audiences=authd user_claim=username token_policies=testuserpolicy

echo Create JWT for authd. Put it in dev/secret/authd.jwt as stated in conf/main.yaml and do chmod 0600.

VAULT_TOKEN=$ENTITY_TOKEN vault read -format=json identity/oidc/token/myrole | jq -r '.data.token'

EOF

#POST /auth/myjwt/login
#  "role": "authd",
#  "jwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
#
#{
#    "auth":{
#        "client_token":"f33f8c72-924e-11f8-cb43-ac59d697597c",
#        "accessor":"0e9e354a-520f-df04-6867-ee81cae3d42d",
#        "policies":[
#            "default",
#            "dev",
#            "prod"
#        ],
#        "lease_duration":2764800,
#        "renewable":true
#    },
#    ...
#}
