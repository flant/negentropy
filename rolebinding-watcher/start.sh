#!/usr/bin/env bash

set -e

function usage {
  echo -e "Usage: $(basename "$0") [option]\n"
  echo "Options:"
  echo "  demo          watch examples if requests at logs of docker container"
  echo "  local         for local runs, replace HTTP_URL with valid url"
}

case "$1" in
  demo)
    MODE="demo"
    QUERY_NAME="demo"
    HTTP_URL=""
    ;;
  local)
    MODE="local"
    QUERY_NAME="rbw"
    HTTP_URL="http://hook-listener:9200/foobar"
    ;;
  --help|-h|"")
    usage && exit 0
    ;;
  *)
    printf "Illegal option $1\n"
    usage && exit 1
    ;;
esac

if ! [[ -f ./id_rsa ]]; then
  ssh-keygen  -t rsa -m PEM -N '' -f ./id_rsa  <<<y >/dev/null 2>&1
  # remove public key in openssh format, because we need rsa format
  rm ./id_rsa.pub
fi

if ! [[ -f ./id_rsa.pub ]]; then
  ssh-keygen  -e -f ./id_rsa  -m PEM > ./id_rsa.pub
fi

if [[ -f /tmp/vaults ]]; then
  ROOT_VAULT_TOKEN=$(cat /tmp/vaults | jq -r '[.[]|select(.name=="root")][0].token')
else
  echo "ERROR: vault token file is missing"
  exit 1
fi

RSA_PRIVATE_KEY=$(cat ./id_rsa | awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}')
RSA_PUBLIC_KEY=$(cat ./id_rsa.pub | awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}')
ROOT_VAULT_PUBLIC_KEY=$(curl -s -k --header "X-Vault-Token: $ROOT_VAULT_TOKEN" https://127.0.0.1:8300/v1/flant/kafka/public_key | jq -r .data.public_key)

curl -k --location --request POST "https://localhost:8300/v1/flant/replica/$QUERY_NAME" \
--header "X-Vault-Token: $ROOT_VAULT_TOKEN" \
--header "Content-Type: application/json" \
--data-raw '{"type":"Vault","send_current_state_at_start":true,"public_key":"'"$RSA_PUBLIC_KEY"'"}' &> /dev/null

export CLIENT_TOPIC=root_source.$QUERY_NAME
export CLIENT_GROUP_ID=$QUERY_NAME
export CLIENT_ENCRYPTION_PUBLIC_KEY=$ROOT_VAULT_PUBLIC_KEY
export CLIENT_ENCRYPTION_PRIVATE_KEY=$RSA_PRIVATE_KEY
export HTTP_URL=$HTTP_URL

docker-compose up -d
