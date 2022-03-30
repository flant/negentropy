storage "file" {
  path = "/tmp/vault/data"
}

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = "true"
  max_request_duration = "120s"
}

api_addr = "http://0.0.0.0:8200"

cluster_name = "auth"

ui = false

log_level = "debug"
