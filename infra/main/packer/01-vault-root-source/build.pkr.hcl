variable "root_password" {
  type =  string
  sensitive = true
}
variable "gcp_vault_bucket" {
  type = string
}
variable "gcp_project" {
  type =  string
}
variable "gcp_zone" {
  type =  string
}
variable "git_directory_checksum" {
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
  default = "2"
}

variable "machine_type" {
  type    = string
  default = "e2-micro"
}

variable "ssh_wait_timeout" {
  type    = string
  default = "90s"
}

locals {
  version_dashed = regex_replace(var.version, "[.]", "-")
  image_name = "${var.name}-${var.git_directory_checksum}"
}

source "googlecompute" "vault-root-source" {
  source_image_family = var.source_image_family

  machine_type        = var.machine_type

  ssh_username        = "root"
  ssh_password        = var.root_password

  disk_size         = var.disk_size
  image_description = "Vault Root Source ${var.version} based on Alpine Linux x86_64 Virtual"
  image_family      = var.name
  image_labels = {
    git_directory_checksum = var.git_directory_checksum,
    version = local.version_dashed
  }
  image_name          = local.image_name
  project_id          = var.gcp_project

  ssh_wait_timeout    = var.ssh_wait_timeout
  zone                = var.gcp_zone
}

build {
  sources = ["source.googlecompute.vault-root-source"]

  provisioner "shell" {
    execute_command = "/bin/sh -x '{{ .Path }}'"
    scripts         = [
      "scripts/00-apk.sh",
      "scripts/02-vault.sh",
      "scripts/03-vector-enable.sh",
      "scripts/80-read-only.sh",
      "scripts/90-cleanup.sh",
      "scripts/91-minimize.sh"
    ]
  }

  provisioner "file" {
    source      = "config/vault.hcl"
    destination = "/etc/vault.hcl.tpl"
  }

  provisioner "shell" {
    environment_vars = [
      "GCP_VAULT_BUCKET=${var.gcp_vault_bucket}"
    ]
    inline = ["envsubst < /etc/vault.hcl.tpl > /etc/vault.hcl"]
  }

  provisioner "shell" {
    execute_command = "/bin/sh -x '{{ .Path }}'"
    scripts         = [
      "scripts/99-sshd.sh"
    ]
  }
}
