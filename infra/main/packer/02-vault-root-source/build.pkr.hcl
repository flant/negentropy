variable "root_password" {
  type =  string
  sensitive = true
}
variable "gcp_vault_root_source_bucket" {
  type = string
}
variable "gcp_ckms_seal_key_ring" {
  type =  string
}
variable "gcp_ckms_seal_crypto_key" {
  type =  string
}
variable "gcp_project" {
  type =  string
}
variable "gcp_region" {
  type =  string
}
variable "gcp_zone" {
  type =  string
}
variable "tfstate_bucket" {
  type =  string
}
variable "vault_recovery_shares" {
  type =  string
  default = "3"
}
variable "vault_recovery_threshold" {
  type =  string
  default = "2"
}
variable "image_sources_checksum" {
  type    = string
}

variable "source_image_family" {
  type    = string
  default = "alpine-base"
}

variable "name" {
  type    = string
  default = "vault-root-source"
}

variable "version" {
  type    = string
  default = "1.7.0"
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

source "googlecompute" "vault-root-source" {
  source_image_family = local.source_image_family

  machine_type        = var.machine_type

  ssh_username        = "root"
  ssh_password        = var.root_password

  disk_size         = var.disk_size
  image_description = "Vault Root Source ${var.version} based on Alpine Linux x86_64 Virtual"
  image_family      = local.image_family
  image_labels = {
    image_sources_checksum = var.image_sources_checksum,
    version = local.version_dashed
  }
  image_name          = local.image_name
  project_id          = var.gcp_project

  ssh_wait_timeout    = var.ssh_wait_timeout
  zone                = var.gcp_zone
}

build {
  sources = ["source.googlecompute.vault-root-source"]

  provisioner "file" {
    source      = "../../../common/vault/vault/bin/vault"
    destination = "/bin/vault"
  }

  provisioner "file" {
    source      = "../../../common/vault/recovery-pgp-keys"
    destination = "/etc/"
  }

  provisioner "file" {
    source      = "config/vault.hcl"
    destination = "/etc/vault.hcl"
  }

  provisioner "file" {
    source      = "../../../common/config/scripts/vault-variables.sh"
    destination = "/etc/vault-variables.sh"
  }

  provisioner "shell" {
    environment_vars = [
      "GCP_VAULT_ROOT_SOURCE_BUCKET=${var.gcp_vault_root_source_bucket}",
      "GCP_PROJECT=${var.gcp_project}",
      "GCP_REGION=${var.gcp_region}",
      "GCPCKMS_SEAL_KEY_RING=${var.gcp_ckms_seal_key_ring}",
      "GCPCKMS_SEAL_CRYPTO_KEY=${var.gcp_ckms_seal_crypto_key}",
      "TFSTATE_BUCKET=${var.tfstate_bucket}",
      "VAULT_RECOVERY_SHARES=${var.vault_recovery_shares}",
      "VAULT_RECOVERY_THRESHOLD=${var.vault_recovery_threshold}"
    ]
    inline = [
      "tmp=$(mktemp); envsubst < /etc/vault-variables.sh > $tmp && cat $tmp > /etc/vault-variables.sh"
    ]
  }

  provisioner "shell" {
    execute_command = "/bin/sh -x '{{ .Path }}'"
    scripts         = [
      "../../../common/packer-scripts/02-vault.sh",
      "../../../common/packer-scripts/03-vector-enable.sh",
      "../../../common/packer-scripts/80-read-only.sh",
      "../../../common/packer-scripts/90-cleanup.sh",
      "../../../common/packer-scripts/91-minimize.sh",
      "../../../common/packer-scripts/99-sshd.sh"
    ]
  }
}
