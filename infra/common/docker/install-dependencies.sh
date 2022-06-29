#!/usr/bin/env sh

# Base dependencies
apk update
apk add bash curl ca-certificates gnupg jq py3-pip make gcc g++ git gcompat musl-dev libffi-dev python3-dev patch

# Terraform
export TERRAFORM_VERSION=1.1.7
curl -L https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip | unzip -p - > /bin/terraform
chmod +x /bin/terraform

# Packer
export PACKER_VERSION=1.8.0
curl -L https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip | unzip -p - > /bin/packer
chmod +x /bin/packer

# Google Cloud SDK
curl -L https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz | tar -xzC /usr/local && \
/usr/local/google-cloud-sdk/install.sh --quiet && \
ln -s /usr/local/google-cloud-sdk/bin/gcloud /bin/gcloud && \
ln -s /usr/local/google-cloud-sdk/bin/gsutil /bin/gsutil

# Python
export CRC32C_PURE_PYTHON=1
python3 -m pip install -r requirements.txt

# Go
GO_VERSION="1.16"
export GOPATH="/opt/golang"
export GOROOT="$GOPATH/local/go${GO_VERSION}"
mkdir -p $GOROOT
curl -L https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar --strip-components=1 -xzC $GOROOT
export PATH="$PATH:$GOROOT/bin:$GOPATH/bin"

# Google credentials
echo "$GOOGLE_CREDENTIALS" > /tmp/credentials.json
export GOOGLE_APPLICATION_CREDENTIALS="/tmp/credentials.json"
gcloud auth activate-service-account --key-file /tmp/credentials.json
export GOOGLE_PROJECT=$(cat /tmp/credentials.json | jq -r .project_id)
gcloud config set project $GOOGLE_PROJECT

# Get CA
gcloud privateca roots describe negentropy --location=europe-west1 --pool=negentropy-flant-local --format="get(pemCaCertificates)" > /usr/local/share/ca-certificates/negentropy-flant-local.pem && \
update-ca-certificates

# TODO: remove later
git config --global --add safe.directory /app
