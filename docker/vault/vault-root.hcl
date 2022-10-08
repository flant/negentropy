storage "file" {
  path = "/tmp/vault/data"
}

listener "tcp" {
  address = "0.0.0.0:8300"
  tls_disable = "false"
  tls_cert_file = "etc/vault-tls/tls.crt"
  tls_key_file  = "etc/vault-tls/tls.key"
  max_request_duration = "120s"
}

api_addr = "https://0.0.0.0:8300"

cluster_name = "root-source"

ui = false

log_level = "debug"
