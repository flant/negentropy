#!/usr/bin/env bash

# Common variables for all vaults.
export GCP_PROJECT="$GCP_PROJECT"
export GCP_REGION="$GCP_REGION"

export HOSTNAME="$(hostname)"
export INTERNAL_ADDRESS="$(ip r get 1 | awk '{print $7}')"

export GCPCKMS_SEAL_KEY_RING="$GCPCKMS_SEAL_KEY_RING"
export GCPCKMS_SEAL_CRYPTO_KEY="$GCPCKMS_SEAL_CRYPTO_KEY"

export TFSTATE_BUCKET="$TFSTATE_BUCKET"

export VAULT_CA_NAME="$VAULT_CA_NAME"
export VAULT_CA_POOL="$VAULT_CA_POOL"
export VAULT_CA_LOCATION="$VAULT_CA_LOCATION"

export VAULT_CERT_VALIDITY_DAYS="1"
export VAULT_CERT_EXPIRE_SECONDS="82800" # 23 hours

export VAULT_RECOVERY_SHARES="$VAULT_RECOVERY_SHARES"
export VAULT_RECOVERY_THRESHOLD="$VAULT_RECOVERY_THRESHOLD"

export VAULT_ADDR="https://$(ip r get 1 | awk '{print $7}'):443"

export VAULT_ROOT_TOKEN_PGP_KEY="$(hostname)-temporary-pub-key.asc"
export VAULT_ROOT_TOKEN_ENCRYPTED="$(hostname)-root-token"
export VAULT_RECOVERY_KEYS_ENCRYPTED="$(hostname)-recovery-keys"

# All vaults except `vault-auth` have fixed subdomain.
if [[ "$VAULT_INTERNAL_SUBDOMAIN" != "" ]]; then
  export VAULT_INTERNAL_FQDN="$VAULT_INTERNAL_SUBDOMAIN.$VAULT_INTERNAL_ROOT_DOMAIN"
fi

# Vault conf variables.
export VAULT_CONF_BUCKET="$VAULT_CONF_BUCKET"

# Vault conf-conf variables.
export VAULT_CONF_CONF_BUCKET="$VAULT_CONF_CONF_BUCKET"

# Vault root-source variables.
export VAULT_ROOT_SOURCE_BUCKET="$VAULT_ROOT_SOURCE_BUCKET"

# Vault auth variables.
export VAULT_AUTH_BUCKET="$(hostname)$VAULT_AUTH_BUCKET_TRAILER"
# For auth vault no subdomain provided, so we base FQDN on the hostname, replacing dash with a dot.
if [[ "$VAULT_INTERNAL_SUBDOMAIN" == "" ]]; then
  export VAULT_INTERNAL_FQDN="$(hostname | sed 's/-/./g').$VAULT_INTERNAL_ROOT_DOMAIN"
  export VAULT_INTERNAL_DOMAIN="${VAULT_INTERNAL_FQDN#*.}"
  export VAULT_PUBLIC_FQDN="$(hostname | sed 's/-/./g').$VAULT_PUBLIC_ROOT_DOMAIN"
  export VAULT_PUBLIC_DOMAIN="${VAULT_PUBLIC_FQDN#*.}"
fi
