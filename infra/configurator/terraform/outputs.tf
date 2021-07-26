output "instance_private_ip_addresses" {
  value = { "private_static_ip_negentropy-vault-conf" : local.vault_conf_private_static_ip }
}
