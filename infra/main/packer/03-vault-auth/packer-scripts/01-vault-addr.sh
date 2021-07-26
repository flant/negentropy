set -exu

sed -i -r 's~(.+VAULT_ADDR=.+):443~\1:8200~g' /etc/vault-variables.sh
