#!/usr/bin/env bash
# Here we have custom build script to calculate checksum
# for the image in '01-alpine-base' directory instead of current one.

# TODO: make this script more simple

dir_path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
export SCRIPT_PATH="$(dirname "$dir_path")/01-alpine-base"

source ../../../common/build/include.sh

if [ -z "$1" ]; then
  if image_exists; then
    >&2 echo "Image already exists in cloud, skipping build."
    exit 0
  fi
else
  >&2 echo "Force rebuild image."
fi

cat ../../../variables.pkrvars.hcl | grep -Ev 'display|headless|accel' > /tmp/variables.pkrvars.hcl
echo "image_sources_checksum = "\""$(image_sources_checksum)"\""" >> /tmp/variables.pkrvars.hcl

packer build \
  -var-file=../../../variables.pkrvars.hcl \
  build.pkr.hcl
code=$?

rm /tmp/variables.pkrvars.hcl
exit $code
