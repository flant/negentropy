#!/usr/bin/env bash

set -e

PLUGINS_DIR="$(realpath $(dirname "$0"))/vault-plugins"
AUTHD_DIR="$(realpath $(dirname "$0"))/authd"
CLI_DIR="$(realpath $(dirname "$0"))/cli"
SERVER_ACCESSD_DIR="$(realpath $(dirname "$0"))/server-access"
NSS_DIR="$(realpath $(dirname "$0"))/server-access/server-access-nss"

function build_plugin() {
  PLUGIN_NAME="$1"
  local EXTRA_MOUNT

  echo "Building $PLUGIN_NAME"
  if [ $PLUGIN_NAME == "flant_iam_auth" ]; then
    EXTRA_MOUNT="-v $PLUGINS_DIR/flant_iam:/go/src/app/flant_iam"
  fi
  if [ $PLUGIN_NAME == "flant_flow" ]; then
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

function build_authd() {
  echo "Building authd"

  docker run --rm \
    -w /go/src/app/authd \
    -v $AUTHD_DIR/build:/src/build \
    -v $AUTHD_DIR:/go/src/app/authd \
    -v /tmp/authd-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16 \
    go build -o /src/build/authd cmd/authd/main.go
}

function build_cli() {
  echo "Building cli"

  docker run --rm \
    -w /go/src/app/cli \
    -v $PLUGINS_DIR:/go/src/app/vault-plugins \
    -v $CLI_DIR/build:/src/build \
    -v $CLI_DIR:/go/src/app/cli \
    -v /tmp/cli-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16 \
    go build -o /src/build/cli cmd/cli/main.go
}

function build_server_accessd() {
  echo "Building server-accessd"

  docker run --rm \
    -w /go/src/server-accessd \
    -v $SERVER_ACCESSD_DIR/flant-server-accessd/build:/src/build \
    -v $SERVER_ACCESSD_DIR:/go/src/server-accessd \
    -v /tmp/server-accessd-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16 \
    go build -o /src/build/server-accessd flant-server-accessd/cmd/main.go
}

function build_nss() {
  echo "Building nss"

  docker run --rm \
    -w /app \
    -v $NSS_DIR:/app \
    rust:1.54 \
    bash -c "cargo build --lib --release --features dynamic_paths && \
             strip -s target/release/libnss_flantauth.so && \
             cp target/release/libnss_flantauth.so build/libnss_flantauth.so.2"
}

mkdir -p $PLUGINS_DIR/build
mkdir -p /tmp/vault-build

mkdir -p $AUTHD_DIR/build
mkdir -p /tmp/authd-build

mkdir -p $CLI_DIR/build
mkdir -p /tmp/cli-build

mkdir -p $SERVER_ACCESSD_DIR/flant-server-accessd/build
mkdir -p /tmp/server-accessd-build

mkdir -p $NSS_DIR/build

plugins=(flant_iam flant_iam_auth flant_flow)

for i in "${plugins[@]}"
do
  build_plugin "$i"
done

build_authd

build_cli

build_server_accessd

build_nss
