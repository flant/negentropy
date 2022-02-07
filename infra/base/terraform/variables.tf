data "google_project" "project" {}

locals {
  google_project_id = data.google_project.project.project_id
  prefix            = "negentropy"
  region            = "europe-west1"
  zone_suffix       = "b"
  ip_cidr_range     = "10.20.254.0/24"
  root_disk_size_gb = "30"
  instance_image    = "ubuntu-2004-focal-v20220110"
  machine_type      = "n1-standard-1"
  ssh_user          = "ubuntu"
  ssh_public_key    = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTXjTmx3hq2EPDQHWSJN7By1VNFZ8colI5tEeZDBVYAe9Oxq4FZsKCb1aGIskDaiAHTxrbd2efoJTcPQLBSBM79dcELtqfKj9dtjy4S1W0mydvWb2oWLnvOaZX/H6pqjz8jrJAKXwXj2pWCOzXerwk9oSI4fCE7VbqsfT4bBfv27FN4/Vqa6iWiCc71oJopL9DldtuIYDVUgOZOa+t2J4hPCCSqEJK/r+ToHQbOWxbC5/OAufXDw2W1vkVeaZUur5xwwAxIb3wM3WoS3BbwNlDYg9UB2D8+EZgNz1CCCpSy1ELIn7q8RnrTp0+H8V9LoWHSgh3VCWeW8C/MnTW90IR"
}
