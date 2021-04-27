#!/usr/bin/env bash

export SCRIPT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source ../../build/include.sh

if image_exists; then
  >&2 echo "Image already exists, skipping build."
  exit 0
fi

packer build \
  -var-file=../../../variables.pkrvars.hcl \
  -var 'image_sources_checksum='"$(image_sources_checksum)"'' \
  -force \
  build.pkr.hcl
