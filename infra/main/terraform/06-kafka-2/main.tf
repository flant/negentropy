data "google_compute_network" "main" {
  name = "${local.prefix}"
}

data "google_compute_subnetwork" "europe-west2" {
  name   = "${local.prefix}-europe-west2"
  region = "europe-west2"
}

data "google_compute_image" "kafka" {
  family = "kafka"
}

resource "google_compute_disk" "kafka-2" {
  zone     = "europe-west2-c"
  name     = "${local.prefix}-kafka-data-2"
  type     = "pd-ssd"
  size     = "30"
}

data "google_service_account" "negentropy-base" {
  account_id = "negentropy-base"
}

resource "google_compute_instance" "kafka-2" {
  zone         = "europe-west2-c"
  name         = "${local.prefix}-kafka-2"
  machine_type = "n1-standard-1"
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = data.google_compute_image.kafka.self_link
      size  = "30"
    }
  }
  attached_disk {
    source      = google_compute_disk.kafka-2.self_link
    device_name = google_compute_disk.kafka-2.name
  }
  network_interface {
    subnetwork = data.google_compute_subnetwork.europe-west2.self_link
    network_ip = local.private_static_ip
  }
  desired_status            = "RUNNING"
  hostname                  = "${local.prefix}-kafka-2.negentropy.flant.local"
  allow_stopping_for_update = true
  metadata = {
    block-project-ssh-keys = "TRUE"
  }
  tags = ["${local.prefix}-kafka-2"]
  service_account {
    email  = data.google_service_account.negentropy-base.email
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_firewall" "kafka-2" {
  name    = "${local.prefix}-kafka-2"
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
    ports    = ["9092", "9093", "7073", "7074", "2182", "2888", "3888"]
  }

  target_tags = ["${local.prefix}-kafka-2"]
}

resource "google_dns_record_set" "kafka-2" {
  name         = "${local.prefix}-kafka-2.negentropy.flant.local."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy-flant-local"
  rrdatas      = [local.private_static_ip]
}

resource "google_dns_record_set" "kafka-2-ptr" {
  name         = "${local.private_ptr}.in-addr.arpa."
  type         = "PTR"
  ttl          = 300
  managed_zone = "ptr"
  rrdatas      = ["${local.prefix}-kafka-2.negentropy.flant.local."]
}
