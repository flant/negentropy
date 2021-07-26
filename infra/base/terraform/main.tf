resource "google_compute_network" "main" {
  name                    = "negentropy"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "bastion" {
  region        = local.region
  name          = "${local.prefix}-bastion"
  network       = google_compute_network.main.self_link
  ip_cidr_range = local.ip_cidr_range
}

resource "google_compute_address" "bastion" {
  name   = "${local.prefix}-bastion"
  region = local.region
}

resource "google_compute_instance" "bastion" {
  zone         = join("-", [local.region, local.zone_suffix])
  name         = "${local.prefix}-bastion"
  machine_type = local.machine_type
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = local.instance_image
      size  = local.root_disk_size_gb
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

  target_tags = ["${local.prefix}-bastion"]
}

resource "google_privateca_certificate_authority" "vault-ca" {
  # TODO: change certificate_authority_id to "local.prefix + name"
  certificate_authority_id = "vault-ca"
  location = "europe-west1"
  pool = "negentropy-flant-local"
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
