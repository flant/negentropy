set -exu

export PKR_VAR_root_password=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c ${1:-32}; echo)

cd /tmp/base/packer/01-alpine-base && \
packer build \
  -var-file=/tmp/variables.pkrvars.hcl \
  -force \
  build.pkr.hcl
