storage "gcs" {
  bucket = "$HOSTNAME$GCP_VAULT_AUTH_BUCKET_TRAILER"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 1
}

api_addr = "http://0.0.0.0:8200"
ui = true
