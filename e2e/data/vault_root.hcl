#storage "file" {
#  path = "/tmp/vault/data"
#}

storage "s3" {
  access_key = "minio"
  secret_key = "minio123"
  endpoint = "minio:9000"
  bucket = "vault-root"
  s3_force_path_style = "true"
  disable_ssl = "true"
}

listener "tcp" {
  address         = "0.0.0.0:8200"
  tls_disable     = "true"
}

api_addr = "http://0.0.0.0:8200"

cluster_name = "root-source"

ui = false

log_level = "debug"
