storage "gcs" {
  bucket = "$VAULT_ROOT_SOURCE_BUCKET"
  ha_enabled = "true"
}

listener "tcp" {
  address         = "$INTERNAL_ADDRESS:443"
  cluster_address = "$INTERNAL_ADDRESS:8201"
  tls_cert_file   = "/tmp/vault.crt"
  tls_key_file    = "/tmp/vault.key"
}

api_addr = "https://$INTERNAL_ADDRESS:443"

seal "gcpckms" {
  project = "$GCP_PROJECT"
  region = "$GCP_REGION"
  key_ring = "$GCPCKMS_SEAL_KEY_RING"
  crypto_key = "$GCPCKMS_SEAL_CRYPTO_KEY"
}

ui = false
