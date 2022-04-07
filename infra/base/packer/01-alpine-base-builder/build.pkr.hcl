variable "gcp_project" {
  type = string
}

variable "gcp_zone" {
  type = string
}

variable "gcp_builder_service_account" {
  type = string
}

variable "source_image_family" {
  type    = string
  default = "ubuntu-vmx"
}

variable "disk_size" {
  type    = string
  default = "10"
}

variable "machine_type" {
  type    = string
  default = "n1-standard-1"
}

variable "ssh_wait_timeout" {
  type    = string
  default = "90s"
}

source "googlecompute" "alpine-base-builder" {
  source_image_family = var.source_image_family

  machine_type     = var.machine_type
  min_cpu_platform = "Intel Haswell"

  ssh_username = "root"

  service_account_email = var.gcp_builder_service_account

  disk_size         = var.disk_size
  skip_create_image = true
  project_id        = var.gcp_project

  ssh_wait_timeout = var.ssh_wait_timeout
  zone             = var.gcp_zone
}

build {
  sources = ["source.googlecompute.alpine-base-builder"]

  provisioner "file" {
    source      = "../../../"
    destination = "/tmp/"
  }

  provisioner "file" {
    source      = "/tmp/variables.pkrvars.hcl"
    destination = "/tmp/"
  }

  provisioner "shell" {
    execute_command = "/bin/sh -x '{{ .Path }}'"
    scripts = [
      "packer-scripts/01-packer-build.sh"
    ]
  }
}
