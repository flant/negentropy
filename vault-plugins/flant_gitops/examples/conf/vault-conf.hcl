listener "tcp" {
  address = "127.0.0.1:8201"
  tls_disable = false
  tls_cert_file = "examples/conf/tls.crt"
  tls_key_file  = "examples/conf/tls.key"
}

cluster_name = "conf"

api_addr = "https://0.0.0.0:8201"

log_level = "debug"

telemetry {
  prometheus_retention_time = "0s"
}

storage "inmem" {}