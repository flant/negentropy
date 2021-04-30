#!/usr/bin/env bash

docker-compose up -d
sleep 3

plugins=(flant_iam flant_iam_auth flant_gitops)

function activate_plugin() {
  plugin="$1"

  if [ $plugin == "flant_iam_auth" ]; then
      docker-compose exec vault sh -c "vault auth enable -path=$plugin $plugin"
  else
      docker-compose exec vault sh -c "vault secrets enable -path=$plugin $plugin"
  fi


  if [ $plugin == "flant_iam" ]; then
    docker-compose exec vault sh -c "vault write -force $plugin/kafka/generate_csr" >/dev/null 2>&1
    docker-compose exec vault sh -c "vault write $plugin/kafka/configure_access kafka_endpoints=kafka:9092"
    docker-compose exec vault sh -c "vault write flant_iam/kafka/configure self_topic_name=root_source"
  fi
}

specifiedPlugin=$1

docker-compose exec vault sh -c "vault token create -orphan -policy=root -field=token > /vault/testdata/token"

# single plugin build
if [ -n "$specifiedPlugin" ]; then
  activate_plugin "$specifiedPlugin"
  exit 0;
fi

for i in "${plugins[@]}"
do
	activate_plugin "$i"
done
