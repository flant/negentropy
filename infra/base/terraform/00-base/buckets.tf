resource "google_storage_bucket" "terraform-state" {
  name          = "${local.google_project_id}-terraform-state"
  location      = "EU"
  # TODO: remove before deploy to production
  force_destroy = true
}

resource "google_storage_bucket" "packer" {
  name          = "${local.google_project_id}-packer"
  location      = "EU"
  # TODO: remove before deploy to production
  force_destroy = true
}
