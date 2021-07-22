#!/usr/bin/env bash

set -e

PLUGINS_DIR="$(cd -- "$(dirname "$0")" >/dev/null 2>&1 || exit 2; pwd -P)"

function build_plugin() {
  PLUGIN_NAME="$1"
  local EXTRA_MOUNT

  echo "Building $PLUGIN_NAME"
  if [ $PLUGIN_NAME == "flant_iam_auth" ]; then
    EXTRA_MOUNT="-v $PLUGINS_DIR/flant_iam:/go/src/app/flant_iam"
  fi

  docker run --rm \
    -w /go/src/app/"$PLUGIN_NAME" \
    -v $PLUGINS_DIR/build:/src/build \
    -v $PLUGINS_DIR/"$PLUGIN_NAME":/go/src/app/"$PLUGIN_NAME" \
    -v $PLUGINS_DIR/shared:/go/src/app/shared \
    -v /tmp/vault-build:/go/pkg/mod \
    $EXTRA_MOUNT \
    -e CGO_ENABLED=1 \
    tetafro/golang-gcc:1.16-alpine \
    go build -tags musl -o /src/build/"$PLUGIN_NAME" cmd/"$PLUGIN_NAME"/main.go
}

mkdir -p $PLUGINS_DIR/build
mkdir -p /tmp/vault-build

plugins=(flant_iam flant_iam_auth)

for i in "${plugins[@]}"
do
  build_plugin "$i"
done
