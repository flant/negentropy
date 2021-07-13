storage "gcs" {
  bucket = "$HOSTNAME$VAULT_AUTH_BUCKET_TRAILER"
}

listener "tcp" {
  address         = "127.0.0.1:443"
  tls_cert_file   = "/tmp/vault.crt"
  tls_key_file    = "/tmp/vault.key"
}

api_addr = "https://127.0.0.1:443"

seal "gcpckms" {
  project = "$GCP_PROJECT"
  region = "$GCP_REGION"
  key_ring = "$GCPCKMS_SEAL_KEY_RING"
  crypto_key = "$GCPCKMS_SEAL_CRYPTO_KEY"
}

ui = false
