resource "google_storage_bucket" "vault-root-source" {
  name          = "${local.google_project_id}-vault-root-source"
  location      = "EU"
  # TODO: remove before deploy to production
  force_destroy = true
}
