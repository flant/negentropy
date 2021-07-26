#!/bin/bash

set -e

export VAULT_ADDR="https://root-source.flant-sandbox.flant.com"

if [ -z "$VAULT_ROOT_SOURCE_TOKEN" ]; then
  echo VAULT_ROOT_SOURCE_TOKEN is unset
  exit 1
fi

export VAULT_TOKEN=$VAULT_ROOT_SOURCE_TOKEN

export tenantName="flant"
export projectName="security"

echo create tenant
tenantResponse=$(vault write flant_iam/tenant identifier=$tenantName --format=json)
tenantUUID=$(jq -r '.data.tenant.uuid' <<< $tenantResponse)

echo create project
projectResponse=$(vault write flant_iam/tenant/$tenantUUID/project identifier=$projectName --format=json)
projectUUID=$(jq -r '.data.project.uuid' <<< $projectResponse)

echo create user
userResponse=$(vault write flant_iam/tenant/$tenantUUID/user identifier=anton.vinogradov first_name=Anton last_name=Vinogradov email=anton.vinogradov@flant.com --format=json)
userUUID=$(jq -r '.data.user.uuid' <<< "$userResponse")

echo create group
echo '{ "identifier": "servers/'$tenantName'", "members": [{"type": "user", "uuid": "'$userUUID'"}] }' > /tmp/gdata.json
groupResponse=$(vault write flant_iam/tenant/$tenantUUID/group @/tmp/gdata.json --format=json)

echo create roles
vault write flant_iam/role name=ssh scope=project --format=json &> /dev/null
vault write flant_iam/role name=servers scope=tenant --format=json &> /dev/null

echo create role binding
echo '{"identifier": "test", "any_project": true, "members":[{"type": "user", "uuid": "'$userUUID'"}], "roles": [{"name": "ssh", "options": {}}], "ttl": 1000000 }' > /tmp/rbdata.json
roleBindingResponse=$(vault write flant_iam/tenant/$tenantUUID/role_binding @/tmp/rbdata.json --format=json)
roleBindingUUID=$(jq -r '.data.role_binding.uuid' <<< "$roleBindingResponse")

echo create servers
#serverClientFull=$(vault write flant_iam/tenant/$tenantUUID/project/$projectUUID/register_server identifier=test-client --format=json)
#serverClientUUID=$(echo $serverClientFull | jq -r '.data.uuid')
#serverClientJWT=$(echo $serverClientFull | jq -r '.data.multipassJWT')

serverServerFull=$(vault write flant_iam/tenant/$tenantUUID/project/$projectUUID/register_server identifier=soc2-report-generator --format=json)
serverServerUUID=$(echo $serverServerFull | jq -r '.data.uuid')
serverServerJWT=$(echo $serverServerFull | jq -r '.data.multipassJWT')

echo create connection info
#vault write flant_iam/tenant/$tenantUUID/project/$projectUUID/server/$serverClientUUID/connection_info hostname=test-client --format=json
vault write flant_iam/tenant/$tenantUUID/project/$projectUUID/server/$serverServerUUID/connection_info hostname=95.217.82.148 --format=json

echo create multipass
vault write flant_iam/tenant/$tenantUUID/user/$userUUID/multipass ttl=100000 max_ttl=1000000 description=test allowed_roles=ssh --format=json

echo ""
echo DEBUG: tenantUUID is $tenantUUID
echo DEBUG: projectUUID is $projectUUID
echo DEBUG: userUUID is $userUUID
echo DEBUG: roleBindingUUID is $roleBindingUUID
echo DEBUG: serverClientUUID is $serverClientUUID
echo DEBUG: serverClientJWT is $serverClientJWT
echo DEBUG: serverServerUUID is $serverServerUUID
echo DEBUG: serverServerJWT is $serverServerJWT
