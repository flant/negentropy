variable "root_password" {
  type      = string
  sensitive = true
}
variable "gcp_project" {
  type = string
}
variable "gcp_zone" {
  type = string
}
variable "git_directory_checksum" {
  type = string
}

variable "source_image_family" {
  type    = string
  default = "ubuntu-1604-lts"
}

variable "name" {
  type    = string
  default = "ubuntu-vmx"
}

variable "version" {
  type    = string
  default = "16.04"
}

variable "disk_size" {
  type    = string
  default = "10"
}

variable "machine_type" {
  type    = string
  default = "e2-small"
}

variable "ssh_wait_timeout" {
  type    = string
  default = "90s"
}

locals {
  version_dashed = regex_replace(var.version, "[.]", "-")
  image_name = "${var.name}-${var.git_directory_checksum}"
}

source "googlecompute" "ubuntu-vmx" {
  source_image_family = var.source_image_family

  machine_type = var.machine_type

  ssh_username = "root"

  disk_size         = var.disk_size
  image_description = "Ubuntu ${var.version} with vmx license enabled"
  image_family      = var.name
  image_labels = {
    git_directory_checksum = var.git_directory_checksum,
    version                = local.version_dashed
  }
  image_licenses = ["projects/vm-options/global/licenses/enable-vmx"]
  image_name     = local.image_name
  project_id     = var.gcp_project

  ssh_wait_timeout = var.ssh_wait_timeout
  zone             = var.gcp_zone
}

build {
  sources = ["source.googlecompute.ubuntu-vmx"]

  provisioner "shell" {
    execute_command = "/bin/sh -x '{{ .Path }}'"
    scripts = [
      "scripts/01-packer-qemu.sh"
    ]
  }
}
