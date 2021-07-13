#!/usr/bin/env bash

function connect_plugins() {
  # initilize flant_iam
  docker-compose exec -T vault sh -c "vault write -force flant_iam/kafka/generate_csr" >/dev/null 2>&1
  docker-compose exec -T vault sh -c "vault write flant_iam/kafka/configure_access kafka_endpoints=kafka:9092"
  docker-compose exec -T vault sh -c "vault write flant_iam/kafka/configure self_topic_name=root_source"
  root_pubkey=$(docker-compose exec -T vault sh -c "vault read flant_iam/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')

  # initialize flant_iam_auth
  docker-compose exec -T vault sh -c "vault write -force auth/flant_iam_auth/kafka/generate_csr" >/dev/null 2>&1
  docker-compose exec -T vault sh -c "vault write auth/flant_iam_auth/kafka/configure_access kafka_endpoints=kafka:9092"
   # link replica
  docker-compose exec -T vault sh -c \
    "vault write auth/flant_iam_auth/kafka/configure self_topic_name=auth-source.auth-1 root_topic_name=root_source.auth-1 root_public_key=\"$root_pubkey\""
  auth_pubkey=$(docker-compose exec -T vault sh -c "vault read auth/flant_iam_auth/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')


  # create replica
  docker-compose exec -T vault sh -c "vault write flant_iam/replica/auth-1 type=Vault public_key=\"$auth_pubkey\""

  echo "Connected"
}

function activate_plugin() {
  plugin="$1"

  if [ $plugin == "flant_iam_auth" ]; then
      docker-compose exec -T vault sh -c "vault auth enable -path=$plugin $plugin"
  else
      docker-compose exec -T vault sh -c "vault secrets enable -path=$plugin $plugin"
  fi
}

function user_example() {
#  unset VAULT_TOKEN
#  export VAULT_ADDR="http://127.0.0.1:8200"
#  vault token create -orphan -policy=root -field=token > /tmp/token_aaaa
#  export VAULT_TOKEN="$(cat /tmp/token_aaaa)"
#
#  vault auth enable approle
#  vault policy write good good.hcl
#  vault write auth/approle/role/good secret_id_ttl=30m token_ttl=900s token_policies=good
#  vault write auth/flant_iam_auth/configure_vault_access \
#	  vault_api_url="http://127.0.0.1:8200" \
#	  vault_api_host="vault_host" \
#	  role_name="good" \
#	  secret_id_ttl="120m" \
#	  approle_mount_point="/auth/approle/" \
#	  secret_id="$(vault write -format=json -f auth/approle/role/good/secret-id | jq -r '.data.secret_id')" \
#	  role_id="$(vault read -format=json auth/approle/role/good/role-id | jq -r '.data.role_id')" \
#	  vault_api_ca=""
#
#  vault write auth/flant_iam_auth/auth_source/a \
#    jwt_validation_pubkeys="$(cat pub.keys)" \
#    bound_issuer=http://vault.example.com/ \
#    entity_alias_name=uuid \
#		allow_service_accounts=false
#
#  vault write flant_iam/tenant/$tnu/user identifier=vasya
#  tnu=$(vault write flant_iam/tenant identifier=tudasuda | grep "^uuid" | awk '{print $2}' | cut -c -36)
#  tnu=$(docker-compose exec -T vault sh -c "vault write flant_iam/tenant identifier=tudasuda" | grep "^uuid" | awk '{print $2}' | cut -c -36)
#  docker-compose exec -T vault sh -c "vault write flant_iam/tenant/$tnu/user identifier=vasya"
  echo "DA"
}

specified_plugin=""
if [ "$1" == "connect_plugins" ]; then
  connect_plugins
  exit 0;
elif [ "$1" == "user_example" ]; then
  user_example
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


#docker-compose exec vault sh -c "vault write auth/flant_iam_auth/kafka/configure self_topic_name=auth-source.testme root_topic_name=root_source.auth-testme"
