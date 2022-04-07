data "google_compute_network" "main" {
  name = "${local.prefix}"
}

data "google_compute_subnetwork" "europe-west3" {
  name   = "${local.prefix}-europe-west3"
  region = "europe-west3"
}

data "google_compute_image" "vault-conf-conf" {
  family = "vault-conf-conf"
}

data "google_service_account" "negentropy-base" {
  account_id   = "negentropy-base"
}

resource "google_compute_instance" "vault-conf-conf" {
  zone         = "europe-west3-b"
  name         = "${local.prefix}-vault-conf-conf"
  machine_type = "n1-standard-1"
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = data.google_compute_image.vault-conf-conf.self_link
      size  = "30"
    }
  }
  network_interface {
    subnetwork = data.google_compute_subnetwork.europe-west3.self_link
    network_ip = local.private_static_ip
  }
  desired_status            = "RUNNING"
  hostname                  = "${local.prefix}-vault-conf-conf.negentropy.flant.local"
  allow_stopping_for_update = true
  metadata = {
    block-project-ssh-keys = "TRUE"
  }
  tags = ["${local.prefix}-vault-conf-conf"]
  service_account {
    email  = data.google_service_account.negentropy-base.email
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_firewall" "vault-conf-conf" {
  name    = "${local.prefix}-vault-conf-conf"
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
    ports    = ["8200"]
  }

  target_tags = ["${local.prefix}-vault-conf-conf"]
}

resource "google_dns_record_set" "vault-conf-conf" {
  name         = "conf-conf.negentropy.flant.local."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy-flant-local"
  rrdatas      = [local.private_static_ip]
}
