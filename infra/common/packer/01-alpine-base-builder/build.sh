#!/usr/bin/env bash
dir_path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
export SCRIPT_PATH="$(dirname "$dir_path")/01-alpine-base"
source ../../build/include.sh

if image_exists; then
  >&2 echo "Image already exists, skipping build."
  exit 0
fi

cat ../../../variables.pkrvars.hcl | grep -Ev 'display|headless|accel' > /tmp/variables.pkrvars.hcl
echo "image_sources_checksum = "\""$(image_sources_checksum)"\""" >> /tmp/variables.pkrvars.hcl

packer build \
  -var-file=../../../variables.pkrvars.hcl \
  build.pkr.hcl

rm /tmp/variables.pkrvars.hcl
