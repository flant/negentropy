listener "tcp" {
  address = "0.0.0.0:8205"
  tls_disable = false
  tls_cert_file = "etc/vault/tls.crt"
  tls_key_file  = "etc/vault/tls.key"
}

cluster_name = "root"

api_addr = "https://0.0.0.0:8205"

log_level = "debug"

telemetry {
 prometheus_retention_time = "0s"
}

storage "inmem" {}

disable_mlock = true
