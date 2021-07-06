#!/bin/bash

export PATH=$PATH:/usr/local/go/bin:/root/go/bin
git clone https://github.com/hashicorp/vault.git
pushd vault
git checkout v1.7.1
git reset --hard
rm -rf negentropy
mkdir -p negentropy
cp -R ../../../../vault-plugins negentropy
patch -p1 < ../001_bucket_count.patch
# patch -p1 < ../002_add_flant_plugins.patch
go mod tidy
go mod download
go get github.com/mitchellh/gox
CGO_ENABLED=1 XC_OS="linux" XC_ARCH="amd64" XC_OSARCH="linux/amd64" make dev
popd
