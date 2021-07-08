storage "gcs" {
  bucket = "$GCP_VAULT_CONF_CONF_BUCKET"
}

listener "tcp" {
  address         = "$INTERNAL_ADDRESS:8200"
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
