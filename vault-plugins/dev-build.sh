#!/usr/bin/env bash

SCRIPTDIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 || exit 2 ; pwd -P )"

plugins=(flant_iam flant_iam_auth flant_gitops)

function build_plugin() {
  PLUGIN_NAME="$1"
  local EXTRA_MOUNT

  echo "Building $PLUGIN_NAME"
  if [ $PLUGIN_NAME == "flant_iam_auth" ]; then
    EXTRA_MOUNT="-v $SCRIPTDIR/flant_iam:/go/src/app/flant_iam"
  fi

  docker run --rm \
    -w /go/src/app/"$PLUGIN_NAME" \
    -v $SCRIPTDIR/build:/src/build \
    -v $SCRIPTDIR/$PLUGIN_NAME:/go/src/app/$PLUGIN_NAME \
    -v $SCRIPTDIR/shared:/go/src/app/shared \
    -v /tmp/vault-build:/go/pkg/mod \
    $EXTRA_MOUNT \
    -e CGO_ENABLED=1 \
    tetafro/golang-gcc:1.16-alpine \
    go build -a -tags musl -o /src/build/$PLUGIN_NAME cmd/$PLUGIN_NAME/main.go
}

mkdir -p $SCRIPTDIR/build
mkdir -p /tmp/vault-build

specifiedPlugin=$1

# single plugin build
if [ -n "$specifiedPlugin" ]; then
  build_plugin "$specifiedPlugin"
  exit 0;
fi

for i in "${plugins[@]}"
do
	build_plugin "$i"
done
