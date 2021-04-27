#!/usr/bin/env bash

set -eo pipefail
set -x

pushd ..
env OS=linux make build
popd

docker stop dev-vault 2>/dev/null || true
docker rm dev-vault 2>/dev/null || true

id=$(docker run \
  --cap-add=IPC_LOCK \
  -d \
  -p 8200:8200 \
  -v "$(pwd)/../build:/vault/plugins" \
  -v "$(pwd)/data:/vault/testdata" \
  --name=dev-vault --rm \
  -e VAULT_API_ADDR=http://127.0.0.1:8200 \
  -e VAULT_ADDR=http://127.0.0.1:8200 \
  -e VAULT_TOKEN=root \
  -e VAULT_LOG_LEVEL=debug \
  vault \
  server -dev -dev-plugin-dir=/vault/plugins -dev-root-token-id=root)

docker exec "$id" sh -c "
sleep 1 \
&& vault secrets enable -path=flant_iam flant_iam \
&& vault token create -orphan -policy=root -field=token > /vault/testdata/token
"

# docker exec -it dev-vault sh

# make enable
# vault path-help flant_iam
# vault read flant_iam/tenant/bcfa63ba-41fd-d33b-c13e-ef03daed0aa2
