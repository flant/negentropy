#!/usr/bin/env bash

set -eou pipefail
set -x

env GOOS=linux GOARCH=amd64 go build -o ./build/flant_iam ./cmd/flant_iam/


docker stop dev-vault 2>/dev/null || true
docker rm   dev-vault 2>/dev/null || true

set -e

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

docker exec -it "$id" sh -c "
sleep 1 \
&& vault secrets enable -path=flant_iam flant_iam \
&& vault token create -orphan -policy=root -field=token > /vault/testdata/token \
&& sh
"
# vault secrets enable -path=flant_iam flant_iam
