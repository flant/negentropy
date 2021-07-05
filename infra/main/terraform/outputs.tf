output "instance_private_ip_addresses" {
  value = { for i in local.instances : "private_static_ip_${i.name}" => i.private_static_ip }
}
