#!/usr/bin/env bash

# Common variables for all vaults.
export GCP_PROJECT="$GCP_PROJECT"
export GCP_REGION="$GCP_REGION"
export GCPCKMS_SEAL_KEY_RING="$GCPCKMS_SEAL_KEY_RING"
export GCPCKMS_SEAL_CRYPTO_KEY="$GCPCKMS_SEAL_CRYPTO_KEY"

export TFSTATE_BUCKET="$TFSTATE_BUCKET"

export VAULT_RECOVERY_SHARES="$VAULT_RECOVERY_SHARES"
export VAULT_RECOVERY_THRESHOLD="$VAULT_RECOVERY_THRESHOLD"

export INTERNAL_ADDRESS="$(ip r get 1 | awk '{print $7}')"
export VAULT_ADDR="http://$(ip r get 1 | awk '{print $7}'):8200"

export VAULT_ROOT_TOKEN_PGP_KEY="$(hostname)-temporary-pub-key.asc"
export VAULT_ROOT_TOKEN_ENCRYPTED="$(hostname)-root-token"
export VAULT_RECOVERY_KEYS_ENCRYPTED="$(hostname)-recovery-keys"

export HOSTNAME="$(hostname)"

# Vault conf variables.
export GCP_VAULT_CONF_BUCKET="$GCP_VAULT_CONF_BUCKET"

# Vault conf-conf variables.
export GCP_VAULT_CONF_CONF_BUCKET="$GCP_VAULT_CONF_CONF_BUCKET"

# Vault root-source variables.
export GCP_VAULT_ROOT_SOURCE_BUCKET="$GCP_VAULT_ROOT_SOURCE_BUCKET"

# Vault auth variables.
export GCP_VAULT_AUTH_BUCKET_TRAILER="$GCP_VAULT_AUTH_BUCKET_TRAILER"
