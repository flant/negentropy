#!/usr/bin/env bash

SCRIPTDIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 || exit 2 ; pwd -P )"

mkdir -p "$SCRIPTDIR/build"
mkdir -p /tmp/vault-build

echo "Building flant_iam"

docker run --rm \
  -v $SCRIPTDIR/build:/src/build \
  -v $SCRIPTDIR/../../../flant_iam:/go/src/app/flant_iam \
  -v $SCRIPTDIR/../../../shared:/go/src/app/shared \
  -v /tmp/vault-build:/go/pkg/mod \
  -w /go/src/app/flant_iam \
  -e CGO_ENABLED=1 \
  tetafro/golang-gcc:1.16-alpine \
  go build -a -tags musl -o /src/build/flant_iam cmd/flant_iam/main.go



echo "Building example_plugin"

docker run --rm \
  -v $SCRIPTDIR/build:/src/build \
  -v $SCRIPTDIR:/go/src/app/for/gomod/example_plugin \
  -v $SCRIPTDIR/../../../shared:/go/src/app/shared \
  -v $SCRIPTDIR/../../../flant_iam:/go/src/app/flant_iam \
  -v /tmp/vault-build:/go/pkg/mod \
  -w /go/src/app/for/gomod/example_plugin \
  -e CGO_ENABLED=1 \
  tetafro/golang-gcc:1.16-alpine \
  go build -a -tags musl -o /src/build/example_plugin cmd/example_plugin/main.go