set -exu

cd /tmp/packer/01-alpine-base && \
packer build \
  -var-file=/tmp/variables.pkrvars.hcl \
  -force \
  build.pkr.hcl
