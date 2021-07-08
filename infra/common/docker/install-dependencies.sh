#!/usr/bin/env bash

export DEBIAN_FRONTEND=noninteractive

apt update -y
apt install -y curl lsb-release software-properties-common apt-transport-https ca-certificates gnupg

curl -fsSL https://apt.releases.hashicorp.com/gpg | apt-key add -
apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"

echo "deb https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -

apt update -y

apt install -y terraform jq python3-pip
python3 -m pip install -r /app/infra/common/docker/requirements.txt

# packer dependencies
apt install -y google-cloud-sdk packer git


GO_VERSION="1.16"
export GOPATH="/opt/golang"
export GOROOT="$GOPATH/local/go${GO_VERSION}"
mkdir -p $GOROOT
curl -L https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar --strip-components=1 -xzC $GOROOT
export PATH="$PATH:$GOROOT/bin:$GOPATH/bin"
