listener "tcp" {
  address = "127.0.0.1:8500"
  tls_disable = false
  tls_cert_file = "conf/server-cert.pem"
  tls_key_file  = "conf/server-key.pem"
  tls_client_ca_file = "conf/ca-cert.pem"
  tls_require_and_verify_client_cert = false
}