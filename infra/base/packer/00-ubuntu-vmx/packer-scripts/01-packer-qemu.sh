set -exu

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y qemu-kvm qemu libvirt-clients
apt-get install -y curl software-properties-common apt-transport-https ca-certificates

curl -fsSL https://apt.releases.hashicorp.com/gpg | apt-key add -
apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
apt-get update && apt-get install -y packer
