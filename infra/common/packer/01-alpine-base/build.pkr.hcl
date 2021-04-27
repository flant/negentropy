variable "root_password" {
  type      = string
  sensitive = true
}
variable "gcp_image_bucket" {
  type = string
}
variable "gcp_project" {
  type = string
}
variable "gcp_zone" {
  type = string
}
variable "image_sources_checksum" {
  type = string
}

variable "iso_checksum" {
  type    = string
  default = "e6bbcab275b704bc6521781f2342fff084700b458711fdf315a5816d9885943c"
}

variable "iso_file" {
  type    = string
  default = "https://dl-cdn.alpinelinux.org/alpine/v3.13/releases/x86_64/alpine-virt-3.13.5-x86_64.iso"
}

variable "name" {
  type    = string
  default = "alpine-base"
}

variable "version" {
  type    = string
  default = "3.13.5"
}

variable "disk_size" {
  type    = string
  default = "2G"
}

variable "cpus" {
  type    = string
  default = "1"
}

variable "memory" {
  type    = string
  default = "1024"
}

variable "accel" {
  default     = "kvm"
  description = "hvf for macOS, kvm for Linux"
}

variable "headless" {
  type    = string
  default = "true"
}

variable "display" {
  type        = string
  default     = "none"
  description = "cocoa for macOS"
}

variable "boot_wait" {
  default     = "20s"
  description = "if no accel, should set at least 30s"
}

variable "ssh_wait_timeout" {
  type    = string
  default = "90s"
}

locals {
  version_dashed = regex_replace(var.version, "[.]", "-")
  image_name = "${var.name}-${var.image_sources_checksum}"
}

source "qemu" "alpine-base" {
  iso_url      = var.iso_file
  iso_checksum = var.iso_checksum

  cpus        = var.cpus
  memory      = var.memory
  display     = var.display
  headless    = var.headless
  accelerator = var.accel

  ssh_username = "root"
  ssh_password = var.root_password

  boot_wait         = var.boot_wait
  boot_key_interval = "10ms"
  boot_command = [
    "root<enter><wait>",
    "ifconfig eth0 up && udhcpc -i eth0<enter><wait5>",
    "wget http://{{ .HTTPIP }}:{{ .HTTPPort }}/answers<enter><wait>",
    "setup-alpine -f answers<enter><wait10>",
    "${var.root_password}<enter><wait>",
    "${var.root_password}<enter>",
    "<wait20s>y<enter><wait30s>",
    "reboot<enter><wait30s>",
    "root<enter><wait>",
    "${var.root_password}<enter><wait>",
    "sed -i 's/^#PermitRootLogin .*/PermitRootLogin yes/g' /etc/ssh/sshd_config<enter>",
    "service sshd restart<enter>",
    "exit<enter>"
  ]

  http_directory = "http"

  disk_size        = var.disk_size
  format           = "raw"
  output_directory = "output"

  shutdown_command = "/sbin/poweroff"

  ssh_wait_timeout = "${var.ssh_wait_timeout}"
  vm_name          = "disk.raw"
}

build {
  sources = ["source.qemu.alpine-base"]

  provisioner "shell" {
    execute_command = "/bin/sh -x '{{ .Path }}'"
    scripts = [
      "../../packer-scripts/00-apk.sh",
      "../../packer-scripts/01-vector.sh",
      "scripts/90-setup.sh"
    ]
  }

  post-processors {
    post-processor "compress" {
      output = "output/disk.raw.tar.gz"
    }
    post-processor "googlecompute-import" {
      bucket            = "${var.gcp_image_bucket}"
      image_description = "Alpine Linux ${var.version} x86_64 Virtual"
      image_family      = var.name
      image_labels = {
        image_sources_checksum = "${var.image_sources_checksum}",
        version                = local.version_dashed
      }
      image_name = local.image_name
      project_id = "${var.gcp_project}"
    }
  }
}
