#!/usr/bin/env bash

set -e

SCRIPTDIR="$(realpath $(dirname "$0"))"

function build_plugin() {
  PLUGIN_NAME="$1"
  local EXTRA_MOUNT

  echo "Building $PLUGIN_NAME"
  if [[ $PLUGIN_NAME == "flant_iam_auth" ]]; then
    EXTRA_MOUNT="-v $SCRIPTDIR/vault-plugins/flant_iam:/go/src/app/flant_iam"
  fi

  mkdir -p $SCRIPTDIR/vault-plugins/build
  mkdir -p /tmp/vault-plugins-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/$PLUGIN_NAME \
    -v $SCRIPTDIR/vault-plugins/build:/src/build \
    -v $SCRIPTDIR/vault-plugins/$PLUGIN_NAME:/go/src/app/$PLUGIN_NAME \
    -v $SCRIPTDIR/vault-plugins/shared:/go/src/app/shared \
    -v /tmp/vault-plugins-build:/go/pkg/mod \
    $EXTRA_MOUNT \
    -e CGO_ENABLED=1 \
    tetafro/golang-gcc:1.17-alpine \
    go build -tags musl -o /src/build/$PLUGIN_NAME cmd/$PLUGIN_NAME/main.go
}

function build_authd() {
  echo "Building authd"

  mkdir -p $SCRIPTDIR/authd/build
  mkdir -p /tmp/authd-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/authd \
    -v $SCRIPTDIR/vault-plugins:/go/src/app/vault-plugins \
    -v $SCRIPTDIR/authd/build:/src/build \
    -v $SCRIPTDIR/authd:/go/src/app/authd \
    -v /tmp/authd-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.17 \
    go build -o /src/build/authd cmd/authd/main.go
}

function build_cli() {
  echo "Building cli"

  mkdir -p $SCRIPTDIR/cli/build
  mkdir -p /tmp/cli-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/cli \
    -v $SCRIPTDIR/vault-plugins:/go/src/app/vault-plugins \
    -v $SCRIPTDIR/authd:/go/src/app/authd \
    -v $SCRIPTDIR/cli/build:/src/build \
    -v $SCRIPTDIR/cli:/go/src/app/cli \
    -v /tmp/cli-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.17 \
    go build -o /src/build/cli cmd/cli/main.go
}

function build_server_accessd() {
  echo "Building server-accessd"

  mkdir -p $SCRIPTDIR/server-access/flant-server-accessd/build
  mkdir -p /tmp/server-accessd-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/server-accessd \
    -v $SCRIPTDIR/authd:/go/src/authd \
    -v $SCRIPTDIR/server-access/flant-server-accessd/build:/src/build \
    -v $SCRIPTDIR/server-access:/go/src/server-accessd \
    -v $SCRIPTDIR/cli:/go/src/cli \
    -v $SCRIPTDIR/vault-plugins:/go/src/vault-plugins \
    -v /tmp/server-accessd-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.17 \
    go build -o /src/build/server-accessd flant-server-accessd/cmd/main.go flant-server-accessd/cmd/initcmd.go
}

function build_nss() {
  echo "Building nss"

  mkdir -p $SCRIPTDIR/server-access/server-access-nss/build

  docker run --rm \
    --platform=linux/amd64 \
    -w /app \
    -v $SCRIPTDIR/server-access/server-access-nss:/app \
    rust:1.54 \
    bash -c "cargo build --lib --release --features dynamic_paths && \
             strip -s target/release/libnss_flantauth.so && \
             cp target/release/libnss_flantauth.so build/libnss_flantauth.so.2"
}

function build_oidc_mock() {
  echo "Building oidc-mock"

  mkdir -p $SCRIPTDIR/e2e/tests/lib/oidc_mock/build
  mkdir -p /tmp/oidc-mock-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/oidc-mock \
    -v $SCRIPTDIR/e2e/tests/lib/oidc_mock/build:/src/build \
    -v $SCRIPTDIR/e2e/tests/lib/oidc_mock:/go/src/oidc-mock \
    -v /tmp/oidc-mock-build:/go/pkg/mod \
    golang:1.17-alpine \
    go build -o /src/build/oidc-mock cmd/server.go
}

function build_kafka_consumer() {
  echo "Building kafka-consumer"

  mkdir -p $SCRIPTDIR/kafka-consumer/build
  mkdir -p /tmp/kafka-consumer-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/kafka-consumer \
    -v $SCRIPTDIR/vault-plugins:/go/src/app/vault-plugins \
    -v $SCRIPTDIR/authd:/go/src/app/authd \
    -v $SCRIPTDIR/cli:/go/src/app/cli \
    -v $SCRIPTDIR/e2e:/go/src/app/e2e \
    -v $SCRIPTDIR/kafka-consumer/build:/src/build \
    -v $SCRIPTDIR/kafka-consumer:/go/src/app/kafka-consumer \
    -v /tmp/kafka-consumer-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.17 \
    go build -o /src/build/consumer cmd/consumer/main.go
}

function build_rolebinding_watcher() {
  echo "Building rolebinding-watcher"

  mkdir -p $SCRIPTDIR/rolebinding-watcher/build
  mkdir -p /tmp/rolebinding-watcher-build

  docker run --rm \
    --platform=linux/amd64 \
    -w /go/src/app/rolebinding-watcher \
    -v $SCRIPTDIR/vault-plugins:/go/src/app/vault-plugins \
    -v $SCRIPTDIR/e2e:/go/src/app/e2e \
    -v $SCRIPTDIR/kafka-consumer:/go/src/app/kafka-consumer \
    -v $SCRIPTDIR/rolebinding-watcher/build:/src/build \
    -v $SCRIPTDIR/rolebinding-watcher:/go/src/app/rolebinding-watcher \
    -v /tmp/rolebinding-watcher-build:/go/pkg/mod \
    -e GO111MODULE=on \
    golang:1.17 \
    go build -o /src/build/rolebinding-watcher cmd/watcher/main.go
}


function build_vault() {

  echo "Building vault"

  docker run --rm \
    --platform=linux/amd64 \
    -w /app/infra/common/vault \
    -v $(pwd):/app \
    $EXTRA_MOUNT \
    golang:1.17.12-alpine \
    sh -c "apk add bash git make musl-dev gcc patch && ./build_vault.sh $ARG"
}

function build_all() {
#
#  plugins=(flant_iam flant_iam_auth)
#  for i in "${plugins[@]}"
#  do
#    build_plugin "$i"
#  done
  build_authd
  build_cli
  build_server_accessd
  build_nss
  build_oidc_mock
  build_kafka_consumer
  build_rolebinding_watcher
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
    rolebinding-watcher)
    TARGET="rolebinding-watcher"
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

if [ "$TARGET" == "rolebinding-watcher" ]; then
  build_rolebinding_watcher
fi

if [ "$TARGET" == "vault" ]; then
  build_vault
fi

if [ -z "$TARGET" ]; then
  build_all
fi
