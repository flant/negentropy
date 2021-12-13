resource "google_storage_bucket" "main" {
  for_each = toset(local.buckets)
  name     = each.value
  location = "EU"
  # TODO: remove before deploy to production
  force_destroy = true
}
