output "bastion_public_ip" {
  value = google_compute_instance.bastion.network_interface.0.access_config.0.nat_ip
}

output "negentropy_public_managed_zone_dns" {
  value = google_dns_managed_zone.negentropy.name_servers
}
