resource "google_storage_bucket" "main" {
  name          = "${local.google_project_id}-terraform-state"
  location      = "EU"
  force_destroy = true
}

resource "google_storage_bucket" "packer" {
  name          = "${local.google_project_id}-packer"
  location      = "EU"
  force_destroy = true
}
