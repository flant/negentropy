data "google_project" "project" {}

variable "vault_root_source_loadbalancer_domain" {
  type = string
}

variable "vault_auth_loadbalancer_domain" {
  type = string
}

locals {
  google_project_id = data.google_project.project.project_id
  prefix            = "negentropy"
}
