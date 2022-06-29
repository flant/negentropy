data "google_compute_network" "main" {
  name = "${local.prefix}"
}

data "google_compute_subnetwork" "europe-west3" {
  name   = "${local.prefix}-europe-west3"
  region = "europe-west3"
}

data "google_compute_image" "vault-root-source" {
  family = "vault-root-source"
}

data "google_service_account" "negentropy-base" {
  account_id   = "negentropy-base"
}

resource "google_compute_instance" "vault-root-source-3" {
  zone         = "europe-west3-b"
  name         = "${local.prefix}-vault-root-source-3"
  machine_type = "n1-standard-1"
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = data.google_compute_image.vault-root-source.self_link
      size  = "30"
    }
  }
  network_interface {
    subnetwork = data.google_compute_subnetwork.europe-west3.self_link
    network_ip = local.private_static_ip
  }
  desired_status            = "RUNNING"
  hostname                  = "${local.prefix}-vault-root-source-3.negentropy.flant.local"
  allow_stopping_for_update = true
  metadata = {
    block-project-ssh-keys = "TRUE"
  }
  tags = ["${local.prefix}-vault-root-source-3"]
  service_account {
    email  = data.google_service_account.negentropy-base.email
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_firewall" "vault-root-source-3" {
  name    = "${local.prefix}-vault-root-source-3"
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
    ports    = ["8200", "8201"]
  }

  target_tags = ["${local.prefix}-vault-root-source-3"]
}
