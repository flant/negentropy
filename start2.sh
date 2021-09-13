#!/usr/bin/env bash

function docker-exec() {
  docker-compose -f docker-compose2.yml exec -e VAULT_TOKEN=${VAULT_TOKEN:-root} -T $1 sh -c "${@:2}"
}

docker run -it --rm -v $(pwd):/app -w /app/infra/common/vault golang:1.16.8-alpine sh -c "apk add bash git make musl-dev gcc patch && ./build_vault.sh"

docker-compose -f docker-compose2.yml up -d

docker-exec "vault_root" "vault status"

docker-exec "vault_auth" "vault status"
