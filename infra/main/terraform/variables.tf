data "google_project" "project" {}

variable "prefix" {
  type    = string
  default = "negentropy"
}

locals {
  google_compute_network_name = "negentropy"

  google_project_id = data.google_project.project.project_id

  regions = sort(distinct([for v in local.instances : v.region]))

  bucket_list = distinct([for i in local.instances : i.bucket])

  region_ip_cidr_ranges_map = {
    "europe-west1" : "10.20.1.0/24"
    "europe-west2" : "10.20.2.0/24"
    "europe-west3" : "10.20.3.0/24"
  }

  common = {
    "instance_root_disk_size_gb" : "30"
    "instance_image" : "ubuntu-2004-focal-v20210415"
    "machine_type" : "n1-standard-2"
    "order_public_static_ip" : false
    "ssh_user" : "user"
    "ssh_public_key" : "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTXjTmx3hq2EPDQHWSJN7By1VNFZ8colI5tEeZDBVYAe9Oxq4FZsKCb1aGIskDaiAHTxrbd2efoJTcPQLBSBM79dcELtqfKj9dtjy4S1W0mydvWb2oWLnvOaZX/H6pqjz8jrJAKXwXj2pWCOzXerwk9oSI4fCE7VbqsfT4bBfv27FN4/Vqa6iWiCc71oJopL9DldtuIYDVUgOZOa+t2J4hPCCSqEJK/r+ToHQbOWxbC5/OAufXDw2W1vkVeaZUur5xwwAxIb3wM3WoS3BbwNlDYg9UB2D8+EZgNz1CCCpSy1ELIn7q8RnrTp0+H8V9LoWHSgh3VCWeW8C/MnTW90IR"
  }

  instances = [
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "vault-conf-conf"])
        "region" : "europe-west3"
        "zone_postfix" : "a"
        "tags" : [join("-", [var.prefix, "vault-conf-conf"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-vault-conf-conf"
        "private_static_ip" : "10.20.3.2"
        "image_family" : "vault-conf-conf"
    }),
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "vault-auth"])
        "region" : "europe-west3"
        "zone_postfix" : "a"
        "tags" : [join("-", [var.prefix, "vault-auth"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-vault-auth"
        "private_static_ip" : "10.20.3.4"
        "image_family" : "vault-auth"
    }),
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "vault-root-source-1"])
        "region" : "europe-west1"
        "zone_postfix" : "c"
        "tags" : [join("-", [var.prefix, "vault-root-source-1"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-vault-root-source"
        "private_static_ip" : "10.20.1.11"
        "image_family" : "vault-root-source"
    }),
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "vault-root-source-2"])
        "region" : "europe-west2"
        "zone_postfix" : "a"
        "tags" : [join("-", [var.prefix, "vault-root-source-2"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-vault-root-source"
        "private_static_ip" : "10.20.2.11"
        "image_family" : "vault-root-source"
    }),
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "vault-root-source-3"])
        "region" : "europe-west3"
        "zone_postfix" : "a"
        "tags" : [join("-", [var.prefix, "vault-root-source-3"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-vault-root-source"
        "private_static_ip" : "10.20.3.11"
        "image_family" : "vault-root-source"
    }),
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "kafka-1"])
        "region" : "europe-west1"
        "zone_postfix" : "c"
        "tags" : [join("-", [var.prefix, "kafka-1"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-negentropy-kafka"
        "private_static_ip" : "10.20.1.31"
        "image_family" : "kafka"
        "additional_disk_name" : join("-", [var.prefix, "kafka-data", "1"])
        "additional_disk_size" : "30"
    }),
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "kafka-2"])
        "region" : "europe-west2"
        "zone_postfix" : "c"
        "tags" : [join("-", [var.prefix, "kafka-2"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-negentropy-kafka"
        "private_static_ip" : "10.20.2.31"
        "image_family" : "kafka"
        "additional_disk_name" : join("-", [var.prefix, "kafka-data", "2"])
        "additional_disk_size" : "30"
    }),
    merge(local.common,
      {
        "name" : join("-", [var.prefix, "kafka-3"])
        "region" : "europe-west3"
        "zone_postfix" : "c"
        "tags" : [join("-", [var.prefix, "kafka-3"])]
        "service_account" : {
          "scopes" : ["cloud-platform"]
        }
        "bucket" : "${local.google_project_id}-negentropy-kafka"
        "private_static_ip" : "10.20.3.31"
        "image_family" : "kafka"
        "additional_disk_name" : join("-", [var.prefix, "kafka-data", "3"])
        "additional_disk_size" : "30"
    })
  ]
}
