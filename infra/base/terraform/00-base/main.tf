resource "google_compute_network" "main" {
  name                    = "${local.prefix}"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "bastion" {
  region        = "europe-west1"
  name          = "${local.prefix}-bastion"
  network       = google_compute_network.main.self_link
  ip_cidr_range = "10.20.254.0/24"
}

resource "google_compute_subnetwork" "europe-west1" {
  region        = "europe-west1"
  name          = "${local.prefix}-europe-west1"
  network       = google_compute_network.main.self_link
  ip_cidr_range = "10.20.1.0/24"
}

resource "google_compute_subnetwork" "europe-west2" {
  region        = "europe-west2"
  name          = "${local.prefix}-europe-west2"
  network       = google_compute_network.main.self_link
  ip_cidr_range = "10.20.2.0/24"
}

resource "google_compute_subnetwork" "europe-west3" {
  region        = "europe-west3"
  name          = "${local.prefix}-europe-west3"
  network       = google_compute_network.main.self_link
  ip_cidr_range = "10.20.3.0/24"
}

resource "google_compute_router" "europe-west1" {
  region  = "europe-west1"
  name    = "${local.prefix}-europe-west1"
  network = google_compute_network.main.self_link
}

resource "google_compute_router" "europe-west2" {
  region  = "europe-west2"
  name    = "${local.prefix}-europe-west2"
  network = google_compute_network.main.self_link
}

resource "google_compute_router" "europe-west3" {
  region  = "europe-west3"
  name    = "${local.prefix}-europe-west3"
  network = google_compute_network.main.self_link
}

resource "google_compute_router_nat" "europe-west1" {
  region                             = "europe-west1"
  name                               = "${local.prefix}-europe-west1"
  router                             = google_compute_router.europe-west1.name
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"
  min_ports_per_vm                   = 1024
  subnetwork {
    name                    = google_compute_subnetwork.europe-west1.self_link
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}

resource "google_compute_router_nat" "europe-west2" {
  region                             = "europe-west2"
  name                               = "${local.prefix}-europe-west2"
  router                             = google_compute_router.europe-west2.name
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"
  min_ports_per_vm                   = 1024
  subnetwork {
    name                    = google_compute_subnetwork.europe-west2.self_link
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}

resource "google_compute_router_nat" "europe-west3" {
  region                             = "europe-west3"
  name                               = "${local.prefix}-europe-west3"
  router                             = google_compute_router.europe-west3.name
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"
  min_ports_per_vm                   = 1024
  subnetwork {
    name                    = google_compute_subnetwork.europe-west3.self_link
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}

resource "google_dns_managed_zone" "local" {
  name        = "negentropy-flant-local"
  dns_name    = "negentropy.flant.local."

  visibility = "private"

  private_visibility_config {
    networks {
      network_url = google_compute_network.main.id
    }
  }
}

resource "google_dns_managed_zone" "ptr" {
  name        = "ptr"
  dns_name    = "10.in-addr.arpa."

  visibility = "private"

  private_visibility_config {
    networks {
      network_url = google_compute_network.main.id
    }
  }
}

resource "google_dns_managed_zone" "negentropy" {
  name        = "negentropy"
  dns_name    = "negentropy.dev.flant.com."
}

resource "google_compute_address" "bastion" {
  name   = "${local.prefix}-bastion"
  region = "europe-west1"
}

resource "google_compute_instance" "bastion" {
  zone         = "europe-west1-b"
  name         = "${local.prefix}-bastion"
  machine_type = "n1-standard-1"
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = "ubuntu-2004-focal-v20220110"
      size  = "30"
    }
  }
  network_interface {
    subnetwork = google_compute_subnetwork.bastion.self_link
    access_config {
      nat_ip = google_compute_address.bastion.address
    }
  }
  desired_status            = "RUNNING"
  allow_stopping_for_update = true
  can_ip_forward            = true
  metadata = {
    ssh-keys = "${local.ssh_user}:${local.ssh_public_key}"
    block-project-ssh-keys = "TRUE"
  }
  tags = ["${local.prefix}-bastion"]
}

resource "google_compute_firewall" "bastion" {
  name    = "${local.prefix}-bastion"
  network = google_compute_network.main.self_link

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]

  target_tags = ["${local.prefix}-bastion"]
}

resource "google_privateca_ca_pool" "main" {
  name     = "negentropy-flant-local"
  location = "europe-west1"
  tier     = "DEVOPS"
}

resource "google_privateca_certificate_authority" "main" {
  certificate_authority_id = "${local.prefix}"
  location = "europe-west1"
  pool = google_privateca_ca_pool.main.name
  # TODO: this is required arguments, so I took they default values from terraform documentation
  config {
    subject_config {
      subject {
        organization = "JSC Flant"
        common_name = "negentropy.flant.local"
      }
    }
    x509_config {
      ca_options {
        is_ca = true
      }
      key_usage {
        base_key_usage {
          cert_sign = true
          crl_sign = true
        }
        extended_key_usage {
          server_auth = false
        }
      }
    }
  }
  key_spec {
    algorithm = "RSA_PKCS1_4096_SHA256"
  }
  ignore_active_certificates_on_deletion = true
}

resource "google_kms_key_ring" "main" {
  name     = "${local.prefix}-vault"
  location = "europe"
}

# TODO: create own crypto key for each vault
resource "google_kms_crypto_key" "main" {
  name     = "vault-unseal"
  key_ring = google_kms_key_ring.main.id

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = "HSM"
  }
}
