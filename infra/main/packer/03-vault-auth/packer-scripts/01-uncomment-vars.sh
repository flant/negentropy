set -exu

sed -i '/^#.*VAULT_INTERNAL_ADDITIONAL_DOMAIN/s/^#//' /etc/vault-variables.sh
sed -i '/^#.*VAULT_EXTERNAL_DOMAIN/s/^#//' /etc/vault-variables.sh
sed -i '/^#.*VAULT_EXTERNAL_ADDITIONAL_DOMAIN/s/^#//' /etc/vault-variables.sh
