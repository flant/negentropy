storage "file" {
  path = "/tmp/vault/data"
}

listener "tcp" {
  address = "0.0.0.0:8300"
  tls_disable = "true"
  max_request_duration = "120s"
}

api_addr = "http://0.0.0.0:8300"

cluster_name = "root-source"

ui = false

log_level = "debug"
