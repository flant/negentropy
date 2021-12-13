storage "gcs" {
  bucket = "$VAULT_BUCKET"
}

listener "tcp" {
  address         = "$INTERNAL_ADDRESS:8200"
  tls_cert_file   = "/tmp/internal.crt"
  tls_key_file    = "/tmp/internal.key"
}

api_addr = "https://$INTERNAL_ADDRESS:8200"

seal "gcpckms" {
  project = "$GCP_PROJECT"
  region = "$GCP_REGION"
  key_ring = "$GCPCKMS_SEAL_KEY_RING"
  crypto_key = "$GCPCKMS_SEAL_CRYPTO_KEY"
}

# TODO: fix name hardcoding
cluster_name = "auth"

ui = false

log_level = "Debug"
