#!/bin/bash

# It's a simple script to obtain a fresh JWT after vault-init-jwt.sh

cat <<'EOF' | docker exec -i "dev-vault" ash -

set -eo pipefail

which jq || apk add jq

ROOT_TOKEN=root

echo Create auth token for entity alias

ENTITY_TOKEN=$(vault write -format=json auth/token/create/myrole \
   policies="testuserpolicy" \
   entity_alias="testuser-alias" |  jq -r '.auth.client_token')

# uncomment for short living token
#vault write identity/oidc/role/myrole key=mykey client_id=authd ttl=30s

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
