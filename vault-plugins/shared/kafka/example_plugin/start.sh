#!/usr/bin/env bash

function connect_plugins() {
  # initialize example_plugin
  docker-compose exec -T vault sh -c "vault write -force example_plugin/kafka/generate_csr" >/dev/null 2>&1
  docker-compose exec -T vault sh -c "vault write example_plugin/kafka/configure_access kafka_endpoints=kafka:9092"
  example_pubkey=$(docker-compose exec -T vault sh -c "vault read example_plugin/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')


  # initilize flant_iam
  docker-compose exec -T vault sh -c "vault write -force flant_iam/kafka/generate_csr" >/dev/null 2>&1
  docker-compose exec -T vault sh -c "vault write flant_iam/kafka/configure_access kafka_endpoints=kafka:9092"
  docker-compose exec -T vault sh -c "vault write flant_iam/kafka/configure self_topic_name=root_source"
  root_pubkey=$(docker-compose exec -T vault sh -c "vault read flant_iam/kafka/public_key" | grep public_key | awk '{$1=""; print $0}' | sed 's/^ *//g')


  # activate example_plugin
  docker-compose exec -T vault sh -c "vault write example_plugin/kafka/configure self_topic_name=extension.self.example_plugin-1 root_public_key=\"$root_pubkey\""

  # register extension plugin
  docker-compose exec -T vault sh -c "vault write flant_iam/extension/example_plugin-1 owned_types=[\"user\"] public_key=\"$example_pubkey\"" >/dev/null 2>&1
}

function activate_plugin() {
  plugin="$1"

  docker-compose exec -T vault sh -c "vault secrets enable -path=$plugin $plugin"
}

function user_example() {
  docker-compose exec -T vault sh -c "vault write example_plugin/user identifier=vasya"
  docker-compose exec -T vault sh -c "vault write example_plugin/user identifier=petya"
  docker-compose exec -T vault sh -c "vault write example_plugin/user identifier=kuzya"
}

docker-compose up -d
sleep 3

plugins=(flant_iam example_plugin)

docker-compose exec -T vault sh -c "vault token create -orphan -policy=root -field=token > /vault/testdata/token"

if [ -n "$specified_plugin" ]; then
  	activate_plugin "$specified_plugin"
  	exit 0;
fi

for i in "${plugins[@]}"
do
	activate_plugin "$i"
done


connect_plugins
user_example
