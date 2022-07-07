#!/usr/bin/env bash

set -e

PLUGINS_DIR="$(realpath $(dirname "$0"))/vault-plugins"
AUTHD_DIR="$(realpath $(dirname "$0"))/authd"
CLI_DIR="$(realpath $(dirname "$0"))/cli"
SERVER_ACCESSD_DIR="$(realpath $(dirname "$0"))/server-access"
NSS_DIR="$(realpath $(dirname "$0"))/server-access/server-access-nss"
OIDC_MOCK_DIR="$(realpath $(dirname "$0"))/e2e/tests/lib/oidc_mock"
KAFKA_CONSUMER_DIR="$(realpath $(dirname "$0"))/kafka-consumer"

function build_plugin() {
  PLUGIN_NAME="$1"
  local EXTRA_MOUNT

  echo "Building $PLUGIN_NAME"
  if [[ $PLUGIN_NAME == "flant_iam_auth" ]]; then
    EXTRA_MOUNT="-v $PLUGINS_DIR/flant_iam:/go/src/app/flant_iam"
  fi

  mkdir -p $PLUGINS_DIR/build
  mkdir -p /tmp/vault-plugins-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/$PLUGIN_NAME \
    -v $PLUGINS_DIR/build:/src/build \
    -v $PLUGINS_DIR/$PLUGIN_NAME:/go/src/app/$PLUGIN_NAME \
    -v $PLUGINS_DIR/shared:/go/src/app/shared \
    -v /tmp/vault-plugins-build:/go/pkg/mod \
    $EXTRA_MOUNT \
    -e CGO_ENABLED=1 \
    tetafro/golang-gcc:1.16-alpine \
    go build -tags musl -o /src/build/$PLUGIN_NAME cmd/$PLUGIN_NAME/main.go
}

function build_authd() {
  echo "Building authd"

  mkdir -p $AUTHD_DIR/build
  mkdir -p /tmp/authd-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/authd \
    -v $PLUGINS_DIR:/go/src/app/vault-plugins \
    -v $AUTHD_DIR/build:/src/build \
    -v $AUTHD_DIR:/go/src/app/authd \
    -v /tmp/authd-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16 \
    go build -o /src/build/authd cmd/authd/main.go
}

function build_cli() {
  echo "Building cli"

  mkdir -p $CLI_DIR/build
  mkdir -p /tmp/cli-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/cli \
    -v $PLUGINS_DIR:/go/src/app/vault-plugins \
    -v $AUTHD_DIR:/go/src/app/authd \
    -v $CLI_DIR/build:/src/build \
    -v $CLI_DIR:/go/src/app/cli \
    -v /tmp/cli-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16 \
    go build -o /src/build/cli cmd/cli/main.go
}

function build_server_accessd() {
  echo "Building server-accessd"

  mkdir -p $SERVER_ACCESSD_DIR/flant-server-accessd/build
  mkdir -p /tmp/server-accessd-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/server-accessd \
    -v $AUTHD_DIR:/go/src/authd \
    -v $SERVER_ACCESSD_DIR/flant-server-accessd/build:/src/build \
    -v $SERVER_ACCESSD_DIR:/go/src/server-accessd \
    -v $CLI_DIR:/go/src/cli \
    -v $PLUGINS_DIR:/go/src/vault-plugins \
    -v /tmp/server-accessd-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16 \
    go build -o /src/build/server-accessd flant-server-accessd/cmd/main.go flant-server-accessd/cmd/initcmd.go
}

function build_nss() {
  echo "Building nss"

  mkdir -p $NSS_DIR/build

  docker run --rm \
    --platform=linux/amd64 \
    -w /app \
    -v $NSS_DIR:/app \
    rust:1.54 \
    bash -c "cargo build --lib --release --features dynamic_paths && \
             strip -s target/release/libnss_flantauth.so && \
             cp target/release/libnss_flantauth.so build/libnss_flantauth.so.2"
}

function build_oidc_mock() {
  echo "Building oidc-mock"

  mkdir -p $OIDC_MOCK_DIR/build
  mkdir -p /tmp/oidc-mock-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/oidc-mock \
    -v $OIDC_MOCK_DIR/build:/src/build \
    -v $OIDC_MOCK_DIR:/go/src/oidc-mock \
    -v /tmp/oidc-mock-build:/go/pkg/mod \
    golang:1.16-alpine \
    go build -o /src/build/oidc-mock cmd/server.go
}

function build_kafka_consumer() {
  echo "Building kafka-consumer"

  mkdir -p $KAFKA_CONSUMER_DIR/build
  mkdir -p /tmp/kafka-consumer-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/kafka-consumer \
    -v $PLUGINS_DIR:/go/src/app/vault-plugins \
    -v $KAFKA_CONSUMER_DIR/build:/src/build \
    -v $KAFKA_CONSUMER_DIR:/go/src/app/authd \
    -v /tmp/kafka-consumer-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16.8-alpine \
    go build -o /src/build/kafka-consumer cmd/consumer/main.go
}

function build_kafka_consumer() {
  echo "Building kafka-consumer"

  mkdir -p $KAFKA_CONSUMER_DIR/build
  mkdir -p /tmp/kafka-consumer-build

  docker run --rm -it \
    --platform=linux/amd64 \
    -w /go/src/app/kafka-consumer \
    -v $PLUGINS_DIR:/go/src/app/vault-plugins \
    -v $KAFKA_CONSUMER_DIR/build:/src/build \
    -v $KAFKA_CONSUMER_DIR:/go/src/app/kafka-consumer \
    -v /tmp/kafka-consumer-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.16 \
    go build -o /src/build/consumer cmd/consumer/main.go
}

function build_vault() {

  echo "Building vault"

  docker run --rm \
    --platform=linux/amd64 \
    -w /app/infra/common/vault \
    -v $(pwd):/app \
    $EXTRA_MOUNT \
    golang:1.16.8-alpine \
    sh -c "apk add bash git make musl-dev gcc patch && ./build_vault.sh $ARG"
}

function build_all() {
  plugins=(flant_iam flant_iam_auth)
  for i in "${plugins[@]}"
  do
    build_plugin "$i"
  done
  build_authd
  build_cli
  build_server_accessd
  build_nss
  build_oidc_mock
  build_vault
}

while [[ $# -gt 0 ]]; do
  case $1 in
    plugins)
    TARGET="plugins"
    break
    ;;
    authd)
    TARGET="authd"
    break
    ;;
    cli)
    TARGET="cli"
    break
    ;;
    server-accessd)
    TARGET="server-accessd"
    break
    ;;
    nss)
    TARGET="nss"
    break
    ;;
    oidc-mock)
    TARGET="oidc-mock"
    break
    ;;
    kafka-consumer)
    TARGET="kafka-consumer"
    break
    ;;
    vault)
    ARG="$2"
    TARGET="vault"
    break
    ;;
    *)
    echo "Unknown parameter $1"; exit 1
    break
    ;;
  esac
done

if [ "$TARGET" == "plugins" ]; then
  plugins=(flant_iam flant_iam_auth)
  for i in "${plugins[@]}"
  do
    build_plugin "$i"
  done
fi

if [ "$TARGET" == "authd" ]; then
  build_authd
fi

if [ "$TARGET" == "cli" ]; then
  build_cli
fi

if [ "$TARGET" == "server-accessd" ]; then
  build_server_accessd
fi

if [ "$TARGET" == "nss" ]; then
  build_nss
fi

if [ "$TARGET" == "oidc-mock" ]; then
  build_oidc_mock
fi

if [ "$TARGET" == "kafka-consumer" ]; then
  build_kafka_consumer
fi

if [ "$TARGET" == "vault" ]; then
  build_vault
fi

if [ -z "$TARGET" ]; then
  build_all
fi
