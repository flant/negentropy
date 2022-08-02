#!/usr/bin/env bash

set -e

SCRIPTDIR="$(realpath ../$(dirname "$0"))"
ROOT_VAULT_TOKEN=$(cat /tmp/vaults | jq -r '[.[]|select(.name=="root")][0].token')
ROOT_VAULT_URL=http://vault-root:8300

docker run --rm \
  --platform=linux/amd64 \
  --name kafka-consumer-e2e \
  --network=negentropy \
  -w /go/src/app/kafka-consumer/e2e \
  -v $SCRIPTDIR/vault-plugins:/go/src/app/vault-plugins \
  -v $SCRIPTDIR/authd:/go/src/app/authd \
  -v $SCRIPTDIR/cli:/go/src/app/cli \
  -v $SCRIPTDIR/e2e:/go/src/app/e2e \
  -v $SCRIPTDIR/kafka-consumer:/go/src/app/kafka-consumer \
  -e ROOT_VAULT_TOKEN=$ROOT_VAULT_TOKEN \
  -e ROOT_VAULT_URL=$ROOT_VAULT_URL \
  golang:1.16 \
  go test -v ./...
