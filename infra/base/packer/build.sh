#!/usr/bin/env bash
source ../../common/build/build.sh

# We should skip '01-alpine-base' due to it will be built by '01-alpine-base-builder'.
# But if you pass '01-alpine-base' as a target it will be built, because `build_images` function respects target.
# TODO: make it possible to specify multiple directories, not only one
SKIP_DIRECTORY="01-alpine-base"
# And we don't want to build vault for base images.
SKIP_VAULT_BUILD="true"

build_images "$@"
