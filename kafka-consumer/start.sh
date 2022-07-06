#!/usr/bin/env bash

if ! [[ -f ./id_rsa ]]; then
  ssh-keygen  -t rsa -m PEM -N '' -f ./id_rsa  <<<y >/dev/null 2>&1
  # remove public key in openssh format, because we need rsa format
  rm ./id_rsa.pub
fi

if ! [[ -f ./id_rsa.pub ]]; then
  ssh-keygen  -e -f ./id_rsa  -m PEM > ./id_rsa.pub
fi

RSA_PRIVATE_KEY=$(cat ./id_rsa | awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}')
RSA_PUBLIC_KEY=$(cat ./id_rsa.pub | awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}')

if [[ -f /tmp/vaults ]]; then
  VAULT_ROOT_TOKEN=$(cat /tmp/vaults | jq -r '[.[]|select(.name=="root")][0].token')
else
  echo "ERROR: vault token file is missing"
  exit 1
fi

NAME=bush

curl --location --request POST "http://localhost:8300/v1/flant/replica/$NAME" \
--header "X-Vault-Token: $VAULT_ROOT_TOKEN" \
--header "Content-Type: application/json" \
--data-raw '{"type":"Vault","public_key":"'"$RSA_PUBLIC_KEY"'"}' &> /dev/null

VAULT_ROOT_PUBLIC_KEY=$(curl -s --header "X-Vault-Token: $VAULT_ROOT_TOKEN" http://127.0.0.1:8300/v1/flant/kafka/public_key | jq -r .data.public_key)

export KAFKA_CFG_SERVERS=localhost:9094
export KAFKA_CFG_USE_SSL=true
export KAFKA_CFG_CA_PATH=../docker/kafka/ca.crt
export KAFKA_CFG_PRIVATE_KEY_PATH=../docker/kafka/client.key
export KAFKA_CFG_PRIVATE_CERT_PATH=../docker/kafka/client.crt
export CLIENT_CFG_TOPIC=root_source.bush
export CLIENT_CFG_GROUP_ID=bush
export CLIENT_CFG_ENCRYPTION_PRIVATE_KEY=$RSA_PRIVATE_KEY
export CLIENT_CFG_ENCRYPTION_PUBLIC_KEY=$VAULT_ROOT_PUBLIC_KEY
export HTTP_URL=http://localhost:9200/asdf

go run cmd/consumer/main.go
