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
}

function user_example() {
  tnu=$(docker-compose exec -T vault sh -c "vault write flant_iam/tenant identifier=tudasuda" | grep "^uuid" | awk '{print $2}' | cut -c -36)
  docker-compose exec -T vault sh -c "vault write flant_iam/tenant/$tnu/user identifier=vasya"
}


if [ "$1" == "connect_plugins" ]; then
  connect_plugins
  exit 0;
elif [ "$1" == "user_example" ]; then
  user_example
  exit 0;
fi

docker-compose up -d
sleep 3

plugins=(flant_iam flant_iam_auth flant_gitops)

function activate_plugin() {
  plugin="$1"

  if [ $plugin == "flant_iam_auth" ]; then
      docker-compose exec -T vault sh -c "vault auth enable -path=$plugin $plugin"
  else
      docker-compose exec -T vault sh -c "vault secrets enable -path=$plugin $plugin"
  fi
}



docker-compose exec -T vault sh -c "vault token create -orphan -policy=root -field=token > /vault/testdata/token"

for i in "${plugins[@]}"
do
	activate_plugin "$i"
done


#docker-compose exec vault sh -c "vault write auth/flant_iam_auth/kafka/configure self_topic_name=auth-source.testme root_topic_name=root_source.auth-testme"
