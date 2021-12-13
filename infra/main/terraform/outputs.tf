output "instance_private_ip_addresses" {
  value = { for i in local.instances : "private_static_ip_${i.name}" => i.private_static_ip }
}

output "instance_public_ip_addresses" {
  value = { for z in google_compute_address.main : "public_static_ip_${z.name}" => z.address }
}

