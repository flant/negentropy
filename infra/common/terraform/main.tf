data "google_project" "project" {}

locals {
  google_project_id = data.google_project.project.project_id
  prefix            = "negentropy-bastion"
  region            = "europe-west3"
  zone_suffix       = "a"
  ip_cidr_range     = "10.20.254.0/24"
  root_disk_size_gb = "30"
  instance_image    = "ubuntu-2004-focal-v20210415"
  machine_type      = "n1-standard-1"
  ssh_user          = "user"
  ssh_public_key    = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTXjTmx3hq2EPDQHWSJN7By1VNFZ8colI5tEeZDBVYAe9Oxq4FZsKCb1aGIskDaiAHTxrbd2efoJTcPQLBSBM79dcELtqfKj9dtjy4S1W0mydvWb2oWLnvOaZX/H6pqjz8jrJAKXwXj2pWCOzXerwk9oSI4fCE7VbqsfT4bBfv27FN4/Vqa6iWiCc71oJopL9DldtuIYDVUgOZOa+t2J4hPCCSqEJK/r+ToHQbOWxbC5/OAufXDw2W1vkVeaZUur5xwwAxIb3wM3WoS3BbwNlDYg9UB2D8+EZgNz1CCCpSy1ELIn7q8RnrTp0+H8V9LoWHSgh3VCWeW8C/MnTW90IR"
}


# # # # # # # # # # # # # # # # # # # # # # #
resource "google_storage_bucket" "main" {
  name     = "${local.google_project_id}-terraform-state"
  location = "EU"
}

resource "google_compute_network" "main" {
  name                    = "negentropy"
  auto_create_subnetworks = false
}
# # # # # # # # # # # # # # # # # # # # # # #


# # # # # # # # # # # # # # # # # # # # # # #
resource "google_storage_bucket" "packer" {
  name     = "${local.google_project_id}-packer"
  location = "EU"
}

resource "google_service_account" "packer" {
  account_id = "negentropy-packer"
}

resource "google_project_iam_member" "packer-service-account-user" {
  project = local.google_project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.packer.email}"
}

resource "google_project_iam_member" "packer-storage-admin" {
  project = local.google_project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.packer.email}"
}

resource "google_project_iam_member" "packer-srorage-object-admin" {
  project = local.google_project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.packer.email}"
}

resource "google_project_iam_member" "packer-compute-storage-admin" {
  project = local.google_project_id
  role    = "roles/compute.storageAdmin"
  member  = "serviceAccount:${google_service_account.packer.email}"
}
# # # # # # # # # # # # # # # # # # # # # # #

resource "google_compute_subnetwork" "main" {
  region        = local.region
  name          = local.prefix
  network       = google_compute_network.main.self_link
  ip_cidr_range = local.ip_cidr_range
}

resource "google_compute_address" "main" {
  name   = local.prefix
  region = local.region
}

resource "google_compute_instance" "main" {
  zone         = join("-", [local.region, local.zone_suffix])
  name         = local.prefix
  machine_type = local.machine_type
  boot_disk {
    initialize_params {
      type  = "pd-ssd"
      image = local.instance_image
      size  = local.root_disk_size_gb
    }
  }
  network_interface {
    subnetwork = google_compute_subnetwork.main.self_link
    access_config {
      nat_ip = google_compute_address.main.address
    }
  }
  desired_status            = "RUNNING"
  allow_stopping_for_update = true
  can_ip_forward            = true
  metadata = {
    ssh-keys = "${local.ssh_user}:${local.ssh_public_key}"
  }
  tags = [local.prefix]
}

output "bastion_public_ip" {
  value = google_compute_instance.main.network_interface.0.access_config.0.nat_ip
}

resource "google_compute_firewall" "main" {
  name    = local.prefix
  network = google_compute_network.main.self_link

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags = [local.prefix]
}
