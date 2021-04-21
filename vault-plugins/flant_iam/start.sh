#!/usr/bin/env bash

set -eou pipefail
set -x

env OS=linux make build

docker stop dev-vault 2>/dev/null || true
docker rm dev-vault 2>/dev/null || true

id=$(docker run \
  --cap-add=IPC_LOCK \
  -d \
  -p 8200:8200 \
  -v $(pwd)/build:/vault/plugins \
  -v $(pwd)/tests/data:/vault/testdata \
  --name=dev-vault --rm \
  -e VAULT_API_ADDR=http://127.0.0.1:8200 \
  -e VAULT_ADDR=http://127.0.0.1:8200 \
  -e VAULT_TOKEN=root \
  vault \
  server -dev -dev-plugin-dir=/vault/plugins -dev-root-token-id=root)

docker exec "$id" sh -c "
sleep 1 \
&& vault secrets enable -path=flant_iam flant_iam \
&& vault token create -orphan -policy=root -field=token > /vault/testdata/token
"

docker logs -f  dev-vault
# docker exec -it dev-vault sh

# make enable
# vault path-help flant_iam
