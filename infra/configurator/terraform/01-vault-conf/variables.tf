data "google_project" "project" {}

locals {
  google_project_id = data.google_project.project.project_id
  prefix            = "negentropy"
  private_static_ip = "10.20.1.2"
}
