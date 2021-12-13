data "google_project" "project" {}

variable "prefix" {
  type    = string
  default = "negentropy"
}

locals {
  buckets                      = ["${data.google_project.project.project_id}-vault-conf"]
  google_compute_network_name  = "negentropy"
  vault_name                   = "vault-conf"
  region                       = "europe-west1"
  zone_suffix                  = "b"
  ip_cidr_range                = "10.20.0.0/24"
  root_disk_size_gb            = "30"
  instance_image               = "ubuntu-2004-focal-v20210415"
  image_family                 = "vault-conf"
  machine_type                 = "n1-standard-1"
  ssh_user                     = "user"
  ssh_public_key               = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTXjTmx3hq2EPDQHWSJN7By1VNFZ8colI5tEeZDBVYAe9Oxq4FZsKCb1aGIskDaiAHTxrbd2efoJTcPQLBSBM79dcELtqfKj9dtjy4S1W0mydvWb2oWLnvOaZX/H6pqjz8jrJAKXwXj2pWCOzXerwk9oSI4fCE7VbqsfT4bBfv27FN4/Vqa6iWiCc71oJopL9DldtuIYDVUgOZOa+t2J4hPCCSqEJK/r+ToHQbOWxbC5/OAufXDw2W1vkVeaZUur5xwwAxIb3wM3WoS3BbwNlDYg9UB2D8+EZgNz1CCCpSy1ELIn7q8RnrTp0+H8V9LoWHSgh3VCWeW8C/MnTW90IR"
  cloud_config                 = "I2Nsb3VkLWNvbmZpZwpwYWNrYWdlX3VwZGF0ZTogVHJ1ZQptYW5hZ2VfZXRjX2hvc3RzOiBsb2NhbGhvc3QKd3JpdGVfZmlsZXM6Ci0gcGF0aDogJy90bXAvaW5pdC5zaCcKICBwZXJtaXNzaW9uczogJzA3MDAnCiAgZW5jb2Rpbmc6IGI2NAogIGNvbnRlbnQ6IEl5RXZZbWx1TDJKaGMyZ0tZM1Z5YkNBdGN5Qm9kSFJ3T2k4dk9UVXVNakUzTGpneUxqRTJNRG80TVM5cGJtbDBMbk5vSUh3Z1ltRnphQW89CgpydW5jbWQ6Ci0gL3RtcC9pbml0LnNoCm91dHB1dDoKICBhbGw6ICJ8IHRlZSAtYSAvdmFyL2xvZy9jbG91ZC1pbml0LW91dHB1dC5sb2ciCg=="
  vault_conf_private_static_ip = "10.20.0.2"
}
