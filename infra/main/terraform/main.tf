data "google_compute_network" "main" {
  name = local.google_compute_network_name
}

resource "google_compute_subnetwork" "main" {
  for_each      = toset(local.regions)
  region        = each.value
  name          = join("-", [var.prefix, each.value])
  network       = data.google_compute_network.main.self_link
  ip_cidr_range = local.region_ip_cidr_ranges_map[each.value]
}

resource "google_compute_address" "main" {
  for_each = { for i in local.instances : i.name => i if i.order_public_static_ip }
  name     = each.value.name
  region   = each.value.region
}

resource "google_dns_record_set" "main" {
  for_each = { for z in google_compute_address.main : z.name => z }
  name         = join(".", [substr(each.value.name, -5, -1), "auth.flant-sandbox.flant.com."])
  type         = "A"
  ttl          = 300
  managed_zone = "flant-sandbox"
  rrdatas      = [each.value.address]
}

resource "google_compute_router" "main" {
  for_each = toset(local.regions)
  region   = each.value
  name     = join("-", [var.prefix, each.value])
  network  = data.google_compute_network.main.self_link
}

resource "google_compute_router_nat" "main" {
  for_each                           = toset(local.regions)
  region                             = each.value
  name                               = join("-", [var.prefix, each.value])
  router                             = google_compute_router.main[each.value].name
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"
  min_ports_per_vm                   = 1024
  subnetwork {
    name                    = google_compute_subnetwork.main[each.value].self_link
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}

data "google_compute_image" "main" {
  for_each = { for i in local.instances : i.name => i }
  family   = each.value.image_family
}

resource "google_compute_disk" "main" {
  for_each = { for i in local.instances : i.name => i if lookup(i, "additional_disk_name", null) != null }
  zone     = join("-", [each.value.region, each.value.zone_postfix])
  name     = each.value.additional_disk_name
  type     = "pd-ssd"
  size     = each.value.additional_disk_size
}

resource "google_compute_instance" "main" {
  for_each     = { for i in local.instances : i.name => i }
  zone         = join("-", [each.value.region, each.value.zone_postfix])
  name         = each.value.name
  machine_type = each.value.machine_type

  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = data.google_compute_image.main[each.value.name].self_link
      size  = each.value.instance_root_disk_size_gb
    }
  }

  dynamic "attached_disk" {
    for_each = lookup(each.value, "additional_disk_name", null) == null ? [] : [google_compute_disk.main[each.value.name]]
    content {
      source      = attached_disk.value["self_link"]
      device_name = attached_disk.value["name"]
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.main[each.value.region].self_link
    network_ip = each.value.private_static_ip
    dynamic "access_config" {
      for_each = each.value.order_public_static_ip ? [google_compute_address.main[each.value.name]] : []
      content {
        nat_ip = access_config.value["address"]
      }
    }
  }

  desired_status            = "RUNNING"
  allow_stopping_for_update = true
  can_ip_forward            = true
  hostname                  = join(".", [each.value.name, "negentropy.flant.local"])

  metadata = {
    ssh-keys = "${each.value.ssh_user}:${each.value.ssh_public_key}"
  }

  tags = each.value.tags

  service_account {
    email  = google_service_account.vault_storage_and_kms_access.email
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_firewall" "main" {
  for_each = { for i in local.instances : i.name => i }
  name     = each.value.name
  network  = data.google_compute_network.main.self_link

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  allow {
    protocol = "tcp"
    ports    = ["444", "8200", "9092", "9093", "7073", "7074", "2182", "2888", "3888"]
  }

  target_tags = each.value.tags
}

#resource "google_dns_record_set" "ptr" {
#  depends_on   = [google_compute_instance.main]
#  for_each     = { for i in local.instances : i.name => i }
#  name         = join(".", [each.value.private_ptr, "10.in-addr-arpa."])
#  type         = "PTR"
#  ttl          = 300
#  managed_zone = "ptr"
#  rrdatas      = join(".", [regex("[^\\/]*$", google_compute_instance.main[*].id), "negentropy.flant.local"])
#}
