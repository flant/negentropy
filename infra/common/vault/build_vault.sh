#!/bin/bash

set -e

while [[ $# -gt 0 ]]; do
  case $1 in
    -f|--force)
    export FORCE="true"
    ;;
  esac
  shift
done

function clone_vault_git_repository() {
  if [[ ! -d "vault" ]]; then
    git clone https://github.com/hashicorp/vault.git
  fi
}

function check_vault_binary_exist() {
  if [[ "$FORCE" != "true" ]]; then
    BINARY="./vault/bin/vault"
    if [[ -f "$BINARY" ]]; then
      >&2 echo "Skipping vault binary build. It already exists at $BINARY"
      exit 0
    fi
  fi
}

export XC_OS="linux"
export XC_ARCH="amd64"
export XC_OSARCH="linux/amd64"
export BUILD_TAGS="vault musl"
export CGO_ENABLED=1

clone_vault_git_repository

check_vault_binary_exist

pushd vault
git checkout v1.7.1
git reset --hard
go mod edit -require github.com/flant/negentropy/vault-plugins/shared@v0.0.1 -replace github.com/flant/negentropy/vault-plugins/shared@v0.0.1=../../../../vault-plugins/shared
go mod edit -require github.com/flant/negentropy/vault-plugins/flant_gitops@v0.0.0 -replace github.com/flant/negentropy/vault-plugins/flant_gitops@v0.0.0=../../../../vault-plugins/flant_gitops
go mod edit -require github.com/flant/negentropy/vault-plugins/flant_iam@v0.0.0 -replace github.com/flant/negentropy/vault-plugins/flant_iam@v0.0.0=../../../../vault-plugins/flant_iam
go mod edit -require github.com/flant/negentropy/vault-plugins/flant_iam_auth@v0.0.0 -replace github.com/flant/negentropy/vault-plugins/flant_iam_auth@v0.0.0=../../../../vault-plugins/flant_iam_auth
patch -p1 < ../001_bucket_count.patch
patch -p1 < ../002_add_flant_plugins.patch
patch -p1 < ../003_add_loading_info_to_cfg.patch
go get k8s.io/client-go@v0.21.1 # TODO: fix this workaround. Next step crashes without installing more recent version of client-go
go mod tidy
make bootstrap
make dev
code=$?
popd

exit $code
