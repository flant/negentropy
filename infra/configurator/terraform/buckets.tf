resource "google_storage_bucket" "vault" {
  for_each = toset(local.bucket_list)
  name     = each.value
  location = "EU"
  # remove before deploy to production
  force_destroy = true
}
