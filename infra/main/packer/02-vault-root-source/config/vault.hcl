storage "gcs" {
  bucket = "$GCP_VAULT_ROOT_SOURCE_BUCKET"
  ha_enabled = "true"
}

listener "tcp" {
  address         = "$INTERNAL_ADDRESS:8200"
  cluster_address = "$INTERNAL_ADDRESS:8201"
  tls_disable     = "true"
}

api_addr = "http://$INTERNAL_ADDRESS:8200"

seal "gcpckms" {
  project = "$GCP_PROJECT"
  region = "$GCP_REGION"
  key_ring = "$GCPCKMS_SEAL_KEY_RING"
  crypto_key = "$GCPCKMS_SEAL_CRYPTO_KEY"
}

ui = false
