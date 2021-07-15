#!/usr/bin/env bash

function connect_plugins() {
  # initilize flant_iam
  docker-compose exec -T vault sh -c "vault write -force flant_iam/kafka/generate_csr" >/dev/null 2>&1
  docker-compose exec -T vault sh -c "vault write flant_iam/kafka/configure_access kafka_endpoints=kafka:9092"
  root_pubkey=$(docker-compose exec -T vault sh -c "vault read flant_iam/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # initialize flant_iam_auth
  docker-compose exec -T vault sh -c "vault write -force auth/flant_iam_auth/kafka/generate_csr" >/dev/null 2>&1
  docker-compose exec -T vault sh -c "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
  auth_pubkey=$(docker-compose exec -T vault sh -c "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  sleep 1

  # configure flant_iam
  docker-compose exec -T vault sh -c "vault write flant_iam/kafka/configure self_topic_name=root_source"

  # configure flant_iam_auth
  docker-compose exec -T vault sh -c \
    "vault write auth/flant_iam_auth/kafka/configure self_topic_name=auth-source.auth-1 root_topic_name=root_source.auth-1 root_public_key=\"$root_pubkey\""


  # create replica
  docker-compose exec -T vault sh -c "vault write flant_iam/replica/auth-1 type=Vault public_key=\"$auth_pubkey\""

  echo "Connected"
}

function jwt_enable() {
    docker-compose exec -T vault sh -c "vault write -force flant_iam/jwt/enable"
    docker-compose exec -T vault sh -c "vault write -force auth/flant_iam_auth/jwt/enable"
}

function fill_test_data() {
  tenantName="1tv"
  projectName="main"
  # create tenant
  tenantResp=$(docker-compose exec -T vault sh -c "vault write flant_iam/tenant identifier=$tenantName --format=json")
  tenantID=$(jq -r '.data.tenant.uuid' <<< "$tenantResp")


  # create project
  projectResp=$(docker-compose exec -T vault sh -c "vault write flant_iam/tenant/$tenantID/project identifier=$projectName --format=json")
  projectID=$(jq -r '.data.project.uuid' <<< "$projectResp")

  # create user
  userResp=$(docker-compose exec -T vault sh -c "vault write flant_iam/tenant/$tenantID/user identifier=vasya first_name=Vasily last_name=Petrov email=vasya@mail.com --format=json")
  userID=$(jq -r '.data.user.uuid' <<< "$userResp")


  # create role
  roleResp=$(docker-compose exec -T vault sh -c "vault write flant_iam/role name=ssh scope=project --format=json")

  # create role binding
  cat <<EOF | docker-compose exec -T vault sh -
  echo '{"subjects":[{"type": "user", "id": "$userID"}], "roles": [], "ttl": 100000 }' > /tmp/data.json
EOF
  roleBindingResp=$(docker-compose exec -T vault sh -c "vault write flant_iam/tenant/$tenantID/role_binding @/tmp/data.json --format=json")
  roleBindingID=$(jq -r '.data.role_binding.uuid' <<< "$roleBindingResp")


  # create multipass
  mpResp=$(docker-compose exec -T vault sh -c "vault write flant_iam/tenant/$tenantID/user/$userID/multipass ttl=100000 max_ttl=1000000 description=test allowed_roles=ssh --format=json")
  jq -r '.' <<< "$mpResp"
}

function activate_plugin() {
  plugin="$1"

  if [ $plugin == "flant_iam_auth" ]; then
      docker-compose exec -T vault sh -c "vault auth enable -path=$plugin $plugin"

  else
      docker-compose exec -T vault sh -c "vault secrets enable -path=$plugin $plugin"
  fi
}

specified_plugin=""
if [ "$1" == "connect_plugins" ]; then
  connect_plugins
  exit 0;
elif [ "$1" == "user_example" ]; then
  user_example
  exit 0;
elif [ "$1" == "fill" ]; then
  fill_test_data
  exit 0;
else
  specified_plugin="$1"
fi

docker-compose up -d
sleep 3

plugins=(flant_iam flant_iam_auth)

docker-compose exec -T vault sh -c "vault token create -orphan -policy=root -field=token > /vault/testdata/token"

if [ -n "$specified_plugin" ]; then
  	activate_plugin "$specified_plugin"
  	exit 0;
fi

for i in "${plugins[@]}"
do
	activate_plugin "$i"
done

jwt_enable
