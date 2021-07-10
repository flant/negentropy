#!/usr/bin/env bash

# Common variables for all vaults.
export GCPCKMS_SEAL_KEY_RING="$GCPCKMS_SEAL_KEY_RING"
export GCPCKMS_SEAL_CRYPTO_KEY="$GCPCKMS_SEAL_CRYPTO_KEY"

export TFSTATE_BUCKET="$TFSTATE_BUCKET"

export VAULT_CA_NAME="20210708-4qi-uyu"
export VAULT_CA_POOL="negentropy-flant-local"
export VAULT_CA_LOCATION="europe-west1"

export VAULT_CERT_VALIDITY_DAYS="1"
export VAULT_CERT_EXPIRE_SECONDS="82800" # 23 hours

export VAULT_RECOVERY_SHARES="$VAULT_RECOVERY_SHARES"
export VAULT_RECOVERY_THRESHOLD="$VAULT_RECOVERY_THRESHOLD"

export VAULT_ADDR="https://$(ip r get 1 | awk '{print $7}'):8200"

export VAULT_ROOT_TOKEN_PGP_KEY="$(hostname)-temporary-pub-key.asc"
export VAULT_ROOT_TOKEN_ENCRYPTED="$(hostname)-root-token"
export VAULT_RECOVERY_KEYS_ENCRYPTED="$(hostname)-recovery-keys"

# Vault conf variables.
export GCP_VAULT_CONF_BUCKET="$GCP_VAULT_CONF_BUCKET"
export VAULT_CONF_DOMAIN="conf.negentropy.flant.local"

# Vault conf-conf variables.
export GCP_VAULT_CONF_CONF_BUCKET="$GCP_VAULT_CONF_CONF_BUCKET"

# Vault root-source variables.
export GCP_VAULT_ROOT_SOURCE_BUCKET="$GCP_VAULT_ROOT_SOURCE_BUCKET"

# Vault auth variables.
export GCP_VAULT_AUTH_BUCKET_TRAILER="$GCP_VAULT_AUTH_BUCKET_TRAILER"
