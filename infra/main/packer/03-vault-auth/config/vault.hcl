storage "gcs" {
  bucket = "$HOSTNAME$GCP_VAULT_AUTH_BUCKET_TRAILER"
}

listener "tcp" {
  address         = "127.0.0.1:8200"
  tls_disable     = "true"
}

api_addr = "http://127.0.0.1:8200"

seal "gcpckms" {
  project = "$GCP_PROJECT"
  region = "$GCP_REGION"
  key_ring = "$GCPCKMS_SEAL_KEY_RING"
  crypto_key = "$GCPCKMS_SEAL_CRYPTO_KEY"
}

ui = false
