#!/usr/bin/env sh

apk update
apk add bash curl ca-certificates gnupg jq py3-pip make gcc g++ git gcompat musl-dev libffi-dev python3-dev patch

# Terraform.
export TERRAFORM_VERSION=1.0.2
curl -L https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip | unzip -p - > /bin/terraform
chmod +x /bin/terraform

# Packer.
export PACKER_VERSION=1.7.3
curl -L https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip | unzip -p - > /bin/packer
chmod +x /bin/packer

# Google Cloud SDK.
curl -L https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz | tar -xzC /usr/local && \
/usr/local/google-cloud-sdk/install.sh --quiet && \
ln -s /usr/local/google-cloud-sdk/bin/gcloud /bin/gcloud && \
ln -s /usr/local/google-cloud-sdk/bin/gsutil /bin/gsutil

export CRC32C_PURE_PYTHON=1
python3 -m pip install -r requirements.txt

GO_VERSION="1.16"
export GOPATH="/opt/golang"
export GOROOT="$GOPATH/local/go${GO_VERSION}"
mkdir -p $GOROOT
curl -L https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar --strip-components=1 -xzC $GOROOT
export PATH="$PATH:$GOROOT/bin:$GOPATH/bin"

echo "$GOOGLE_CREDENTIALS" > /tmp/credentials-tmp.json
gcloud auth activate-service-account --key-file /tmp/credentials-tmp.json
gcloud privateca roots describe vault-ca --location=europe-west1 --pool=negentropy-flant-local --format="get(pemCaCertificates)" > /usr/local/share/ca-certificates/negentropy-flant-local.pem && \
update-ca-certificates
