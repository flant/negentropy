#!/usr/bin/env bash
source ../../common/build/build.sh

SKIP_VAULT_BUILD="true"

build_images "$@"
