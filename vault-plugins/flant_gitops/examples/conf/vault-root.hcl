listener "tcp" {
  address = "127.0.0.1:8203"
  tls_disable = false
  tls_cert_file = "examples/conf/tls.crt"
  tls_key_file  = "examples/conf/tls.key"
}

cluster_name = "root"

api_addr = "https://0.0.0.0:8203"

log_level = "debug"

telemetry {
 prometheus_retention_time = "0s"
}

storage "inmem" {}