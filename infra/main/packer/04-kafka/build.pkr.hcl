variable "gcp_project" {
  type =  string
}

variable "gcp_region" {
  type =  string
}

variable "gcp_zone" {
  type =  string
}

variable "image_sources_checksum" {
  type    = string
}

variable "kafka_ca_name" {
  type =  string
}

variable "kafka_ca_pool" {
  type =  string
}

variable "kafka_ca_location" {
  type =  string
}

variable "kafka_domain" {
  type    = string
}

variable "kafka_server_key_pass" {
  type    = string
  sensitive = true
}

variable "kafka_bucket" {
  type    = string
}

variable "kafka_replicas" {
  type    = string
  default = "3"
  description = "kafka and zookeeper will be configured to this number of replicas $(hostname-[1..n])"
}

variable "kafka_heap_opts" {
  type    = string
  default = "-Xmx2G -Xms2G"
}

variable "zookeeper_heap_opts" {
  type    = string
  default = "-Xmx1G -Xms1G"
}

variable "source_image_family" {
  type    = string
  default = "alpine-base"
}

variable "name" {
  type    = string
  default = "kafka"
}

variable "version" {
  type    = string
  default = "2.8.0"
}

variable "disk_size" {
  type    = string
  default = "5"
}

variable "machine_type" {
  type    = string
  default = "e2-micro"
}

variable "ssh_wait_timeout" {
  type    = string
  default = "90s"
}

variable "env" {
  type    = string
  default = ""
}

locals {
  version_dashed = regex_replace(var.version, "[.]", "-")
  image_family = "${var.name}${var.env}"
  image_name = "${local.image_family}-${var.image_sources_checksum}"
  source_image_family = "${var.source_image_family}${var.env}"
}

source "googlecompute" "kafka" {
  source_image_family = local.source_image_family

  machine_type        = var.machine_type

  ssh_username        = "root"

  disk_size         = var.disk_size
  image_description = "Kafka ${var.version} based on Alpine Linux x86_64 Virtual"
  image_family      = local.image_family
  image_labels = {
    image_sources_checksum = var.image_sources_checksum,
    version = local.version_dashed
  }
  image_name          = local.image_name
  project_id          = var.gcp_project
  zone                = var.gcp_zone

  temporary_key_pair_type = "ed25519"
  ssh_wait_timeout    = var.ssh_wait_timeout
}

build {
  sources = ["source.googlecompute.kafka"]

  provisioner "shell" {
    inline = ["mkdir -p /etc/kafka"]
  }

  provisioner "file" {
    source      = "config/"
    destination = "/etc/kafka/"
  }

  provisioner "file" {
    source      = "../../../common/config/scripts/kafka-variables.sh"
    destination = "/etc/kafka-variables.sh"
  }

  provisioner "shell" {
    environment_vars = [
      "KAFKA_DOMAIN=${var.kafka_domain}",
      "SERVER_KEY_PASS=${var.kafka_server_key_pass}",
      "KAFKA_BUCKET=${var.kafka_bucket}",
      "KAFKA_CA_NAME=${var.kafka_ca_name}",
      "KAFKA_CA_POOL=${var.kafka_ca_pool}",
      "KAFKA_CA_LOCATION=${var.kafka_ca_location}",
      "KAFKA_REPLICAS=${var.kafka_replicas}"
    ]
    inline = [
      "tmp=$(mktemp); envsubst < /etc/kafka-variables.sh > $tmp && cat $tmp > /etc/kafka-variables.sh"
    ]
  }

  provisioner "shell" {
    execute_command = "/bin/sh -x '{{ .Path }}'"
    scripts         = [
      "../../../common/packer-scripts/03-vector-enable.sh",
      "../../../common/packer-scripts/80-read-only.sh",
      "packer-scripts/01-kafka.sh",
      "packer-scripts/02-zookeeper.sh",
      "../../../common/packer-scripts/90-cleanup.sh",
      "../../../common/packer-scripts/91-minimize.sh",
      "../../../common/packer-scripts/99-tfadm.sh"
    ]
  }

  provisioner "shell" {
    environment_vars = [
      "KAFKA_HEAP_OPTS=${var.kafka_heap_opts}",
      "ZOOKEEPER_HEAP_OPTS=${var.zookeeper_heap_opts}"
    ]
    inline = [
      "tmp=$(mktemp); envsubst '$KAFKA_HEAP_OPTS' < /etc/init.d/kafka > $tmp && cat $tmp > /etc/init.d/kafka",
      "tmp=$(mktemp); envsubst '$ZOOKEEPER_HEAP_OPTS' < /etc/init.d/zookeeper > $tmp && cat $tmp > /etc/init.d/zookeeper"
    ]
  }
}
