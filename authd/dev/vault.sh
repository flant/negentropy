#!/bin/bash

if [ "$1" == "enter" ]; then
  docker exec -it "dev-vault" sh
  exit 0
fi

if [ "$1" == "stop" ]; then
  docker stop dev-vault
  docker rm dev-vault
  exit 0
fi

set -e

  #-d \
docker run \
  --cap-add=IPC_LOCK \
  -d \
  -p 8200:8200 \
  --name=dev-vault \
  -e VAULT_API_ADDR=http://127.0.0.1:8200 \
  -e VAULT_ADDR=http://127.0.0.1:8200 \
  -e VAULT_TOKEN=root \
  vault \
  server -dev -dev-root-token-id=root

# -v $(pwd)/plugins:/vault/plugins \
# server -dev -dev-plugin-dir=/vault/plugins -dev-root-token-id=root



