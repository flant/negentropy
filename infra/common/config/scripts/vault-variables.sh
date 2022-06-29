#!/usr/bin/env bash

# Common variables
export GCP_PROJECT="$GCP_PROJECT"
export GCP_REGION="$GCP_REGION"

# Common variables for all vaults.
export INTERNAL_ADDRESS="$(ip r get 1 | awk '{print $7}')"

export VAULT_ADDR="https://$(ip r get 1 | awk '{print $7}'):8200"

export TFSTATE_BUCKET="$TFSTATE_BUCKET"

export VAULT_CA_NAME="$VAULT_CA_NAME"
export VAULT_CA_POOL="$VAULT_CA_POOL"
export VAULT_CA_LOCATION="$VAULT_CA_LOCATION"

export VAULT_CERT_VALIDITY_DAYS="1"
export VAULT_CERT_EXPIRE_SECONDS="82800" # 23 hours

export VAULT_RECOVERY_SHARES="$VAULT_RECOVERY_SHARES"
export VAULT_RECOVERY_THRESHOLD="$VAULT_RECOVERY_THRESHOLD"

if [[ $(hostname) == *-root-source-* ]]; then
    export VAULT_ROOT_TOKEN_PGP_KEY="negentropy-vault-root-source-temporary-pub-key.asc"
    export VAULT_ROOT_TOKEN_ENCRYPTED="negentropy-vault-root-source-root-token"
    export VAULT_RECOVERY_KEYS_ENCRYPTED="negentropy-vault-root-source-recovery-keys"
else
    export VAULT_ROOT_TOKEN_PGP_KEY="$(hostname)-temporary-pub-key.asc"
    export VAULT_ROOT_TOKEN_ENCRYPTED="$(hostname)-root-token"
    export VAULT_RECOVERY_KEYS_ENCRYPTED="$(hostname)-recovery-keys"
fi
export VAULT_BUCKET="$VAULT_BUCKET"

export GCPCKMS_SEAL_KEY_RING="$GCPCKMS_SEAL_KEY_RING"
export GCPCKMS_SEAL_CRYPTO_KEY="$GCPCKMS_SEAL_CRYPTO_KEY"
export GCPCKMS_REGION="$GCPCKMS_REGION"

export VAULT_INTERNAL_DOMAIN="$VAULT_INTERNAL_DOMAIN"
#export VAULT_INTERNAL_ADDITIONAL_DOMAIN="$(hostname | sed 's/.*-//').$VAULT_INTERNAL_DOMAIN"
#export VAULT_EXTERNAL_DOMAIN="$VAULT_EXTERNAL_DOMAIN"
#export VAULT_EXTERNAL_ADDITIONAL_DOMAIN="$(hostname | sed 's/.*-//').$VAULT_EXTERNAL_DOMAIN"
