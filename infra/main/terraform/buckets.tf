# resource "google_storage_bucket" "vault" {
#   for_each = toset(var.bucket_list)
#   name     = each.value
#   location = "EU"
# }
