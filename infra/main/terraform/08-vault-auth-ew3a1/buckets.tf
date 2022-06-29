# TODO: change name to "vault-auth-ew3a1", because each vault must have it's own bucket
resource "google_storage_bucket" "vault-auth" {
  name          = "${local.google_project_id}-vault-auth"
  location      = "EU"
  # TODO: remove before deploy to production
  force_destroy = true
}
