data "google_compute_network" "main" {
  name = "${local.prefix}"
}

data "google_compute_subnetwork" "europe-west3" {
  name   = "${local.prefix}-europe-west3"
  region = "europe-west3"
}

data "google_compute_image" "vault-auth" {
  family = "vault-auth"
}

data "google_service_account" "negentropy-base" {
  account_id   = "negentropy-base"
}

resource "google_compute_address" "vault-auth-ew3a1" {
  name   = "vault-auth-ew3a1"
  region = "europe-west3"
}

resource "google_compute_instance" "vault-auth-ew3a1" {
  zone         = "europe-west3-a"
  name         = "${local.prefix}-vault-auth-ew3a1"
  machine_type = "n1-standard-1"
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = data.google_compute_image.vault-auth.self_link
      size  = "30"
    }
  }
  network_interface {
    subnetwork = data.google_compute_subnetwork.europe-west3.self_link
    network_ip = local.private_static_ip
    access_config {
      nat_ip = google_compute_address.vault-auth-ew3a1.address
    }
  }
  desired_status            = "RUNNING"
  hostname                  = "${local.prefix}-vault-auth-ew3a1.negentropy.flant.local"
  allow_stopping_for_update = true
  metadata = {
    block-project-ssh-keys = "TRUE"
  }
  tags = ["${local.prefix}-vault-auth-ew3a1"]
  service_account {
    email  = data.google_service_account.negentropy-base.email
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_firewall" "vault-auth-ew3a1" {
  name    = "${local.prefix}-vault-auth-ew3a1"
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
    ports    = ["443", "8200"]
  }

  target_tags = ["${local.prefix}-vault-auth-ew3a1"]
}

resource "google_dns_record_set" "vault-auth-ew3a1" {
  name         = "ew3a1.auth.negentropy.dev.flant.com."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy"
  rrdatas      = [google_compute_address.vault-auth-ew3a1.address]
}
