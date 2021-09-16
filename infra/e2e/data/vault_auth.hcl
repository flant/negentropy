storage "file" {
  path = "/mnt/vault/data"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = "true"
}

api_addr = "http://0.0.0.0:8200"

cluster_name = "auth"

ui = false

log_level = "debug"
