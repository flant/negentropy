#!/bin/bash

ci_mode=""
if [[ "$1" == "--ci-mode" ]]; then
  ci_mode="true"
fi

set -eo pipefail

if [[ "${ci_mode}x" == "x" ]]; then
  set -x

  pushd ..
  env OS=linux make build
  popd

  docker stop dev-vault 2>/dev/null || true
  docker rm dev-vault 2>/dev/null || true
fi

id=$(docker run \
  --cap-add=IPC_LOCK \
  -d \
  -p 8200:8200 \
  --name=dev-vault --rm \
  -e VAULT_API_ADDR=http://127.0.0.1:8200 \
  -e VAULT_ADDR=http://127.0.0.1:8200 \
  -e VAULT_TOKEN=root \
  -e VAULT_LOG_LEVEL=debug \
  -e VAULT_DEV_ROOT_TOKEN_ID=root \
  -v "$(pwd)/../build:/vault/plugins" \
  vault:1.7.1 \
  server -dev -dev-plugin-dir=/vault/plugins)

echo "Sleep a second or more ..." && sleep 5 # TODO(nabokihms): use container healthchecks
if [[ "${ci_mode}x" != "x" ]]; then
  docker logs dev-vault 2>&1
fi

docker exec "$id" vault secrets enable -path=flant_iam vault-plugin-flant-iam
mkdir -p data && echo -n "root" > ./data/token

if [[ "${ci_mode}x" == "x" ]]; then
  docker logs -f  dev-vault
fi
# docker exec -it dev-vault sh

# make enable
# vault path-help flant_iam
# vault read flant_iam/tenant/bcfa63ba-41fd-d33b-c13e-ef03daed0aa2
