set -exu

cd /tmp/01-alpine-base && \
packer build \
  -var-file=variables.pkrvars.hcl \
  -force \
  build.pkr.hcl
