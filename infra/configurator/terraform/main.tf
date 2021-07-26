data "google_compute_network" "main" {
  name = local.google_compute_network_name
}

resource "google_compute_subnetwork" "main" {
  region        = local.region
  name          = join("-", [var.prefix, local.vault_name])
  network       = data.google_compute_network.main.self_link
  ip_cidr_range = local.ip_cidr_range
}

resource "google_compute_router" "main" {
  region  = local.region
  name    = join("-", [var.prefix, local.vault_name])
  network = data.google_compute_network.main.self_link
}

resource "google_compute_router_nat" "main" {
  region                             = local.region
  name                               = join("-", [var.prefix, local.vault_name])
  router                             = google_compute_router.main.name
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"
  min_ports_per_vm                   = 1024
  subnetwork {
    name                    = google_compute_subnetwork.main.self_link
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}

data "google_compute_image" "main" {
  family = local.image_family
}

resource "google_compute_instance" "main" {
  zone         = join("-", [local.region, local.zone_suffix])
  name         = join("-", [var.prefix, local.vault_name])
  machine_type = local.machine_type
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = data.google_compute_image.main.self_link
      size  = local.root_disk_size_gb
    }
  }
  network_interface {
    subnetwork = google_compute_subnetwork.main.self_link
    network_ip = local.vault_conf_private_static_ip
  }
  allow_stopping_for_update = true
  desired_status            = "RUNNING"
  can_ip_forward            = true
  metadata = {
    ssh-keys  = "${local.ssh_user}:${local.ssh_public_key}"
    user-data = base64decode(local.cloud_config)
  }
  tags = [join("-", [var.prefix, local.vault_name])]
  service_account {
    email  = google_service_account.terraform.email
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_firewall" "main" {
  name    = join("-", [var.prefix, local.vault_name])
  network = data.google_compute_network.main.self_link

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

  target_tags = [join("-", [var.prefix, local.vault_name])]
}

resource "google_dns_record_set" "main" {
  name         = "conf.negentropy.flant.local."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy-flant-local"
  rrdatas      = [local.vault_conf_private_static_ip]
}
