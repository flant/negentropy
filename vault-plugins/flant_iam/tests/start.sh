#!/bin/bash

ci_mode=""
if [[ "$1" == "--ci-mode" ]]; then
  ci_mode="true"
fi

set -eo pipefail

if [[ "${ci_mode}x" == "x" ]]; then
  # NOT CI
  set -x

  # pushd ..
  # env OS=linux make build

  # rm -rf ../build/*
  # pushd ../..
  # bash ../../dev-build.sh flant_iam
  mkdir -p build
  # cp ../../build/flant_iam ../build/flant_iam

  # popd

  docker stop dev-vault 2>/dev/null || true
  docker rm dev-vault 2>/dev/null || true
fi

id=$(docker run \
  --cap-add=IPC_LOCK \
  -d \
  -p 8200:8200 \
  --name=dev-vault --rm \
  -e VAULT_API_ADDR=http://127.0.0.1:8200 \
  -e VAULT_ADDR=http://127.0.0.1:8200 \
  -e VAULT_TOKEN=root \
  -e VAULT_LOG_LEVEL=debug \
  -e VAULT_DEV_ROOT_TOKEN_ID=root \
  -v "$(pwd)/build:/vault/plugins" \
  -v "$(pwd)/data:/vault/testdata" \
  vault:1.7.1 \
  server -dev -dev-plugin-dir=/vault/plugins)

echo "Sleep a second or more ..." && sleep 5 # TODO(nabokihms): use container healthchecks
if [[ "${ci_mode}x" != "x" ]]; then
  # IN CI
  docker logs dev-vault 2>&1
fi

docker exec "$id" sh -c "
vault secrets enable -path=flant_iam flant_iam \
&& vault token create -orphan -policy=root -field=token > /vault/testdata/token
"

if [[ "${ci_mode}x" == "x" ]]; then
  # NOT CI

  # USAGE:   docker exec dev-vault /remount.sh flant_iam
  remount_script='#!/bin/sh\nvault plugin deregister $1 && vault plugin register -sha256=$(sha256sum /vault/plugins/$1 | cut -d" " -f1) $1 && vault plugin reload -mounts=$1/'
  docker exec dev-vault sh -c "echo -e '$remount_script' > /remount.sh && chmod +x /remount.sh"

  # USAGE:   docker exec gobuild /gobuild.sh flant_iam
  docker run --rm -d --name gobuild \
    -w /go/src/app \
    -v "$PWD/build:/src/build" \
    -v "$PWD/flant_iam:/go/src/app/flant_iam" \
    -v "$PWD/flant_iam_auth:/go/src/app/flant_iam_auth" \
    -v "$PWD/flant_gitopts:/go/src/app/flant_gitopts" \
    -v "$PWD/shared:/go/src/app/shared" \
    -v /tmp/vault-build:/go/pkg/mod \
    -e CGO_ENABLED=1 \
    tetafro/golang-gcc:1.16-alpine \
    sh -c "echo -e '#!/bin/sh\ncd \$1 && go build -tags musl -o /src/build/\$1 cmd/\$1/main.go' > /gobuild.sh && chmod +x /gobuild.sh && sleep infinity"

  docker exec gobuild /gobuild.sh flant_iam
  docker exec dev-vault /remount.sh flant_iam

  # See logs
  docker logs -f dev-vault
fi

# docker exec -it dev-vault sh

# make enable
# vault path-help flant_iam
# vault read flant_iam/tenant/bcfa63ba-41fd-d33b-c13e-ef03daed0aa2

# BIG HELPERS

# docker exec gobuild /gobuild.sh flant_iam
# docker exec dev-vault vault plugin reload -mounts=flant_iam/
# docker exec dev-vault vault plugin reload -plugin flant_iam
# vault plugin deregister flant_iam && vault plugin register  -sha256=$(sha256sum /vault/plugins/flant_iam | cut -d' ' -f1) flant_iam && vault plugin reload -mounts=flant_iam/
