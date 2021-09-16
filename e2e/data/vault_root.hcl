storage "file" {
  path = "/tmp/vault/data"
}

listener "tcp" {
  address         = "0.0.0.0:8200"
  tls_disable     = "true"
}

api_addr = "http://0.0.0.0:8200"

cluster_name = "root-source"

ui = false

log_level = "debug"
