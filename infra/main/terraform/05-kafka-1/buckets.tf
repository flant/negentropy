resource "google_storage_bucket" "kafka" {
  name          = "${local.google_project_id}-kafka"
  location      = "EU"
  # TODO: remove before deploy to production
  force_destroy = true
}
