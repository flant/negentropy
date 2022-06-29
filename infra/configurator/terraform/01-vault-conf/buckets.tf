resource "google_storage_bucket" "vault-conf" {
  name          = "${local.google_project_id}-vault-conf"
  location      = "EU"
  # TODO: remove before deploy to production
  force_destroy = true
}
