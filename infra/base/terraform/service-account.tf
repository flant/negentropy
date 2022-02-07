# TODO: cleanup roles, maybe there are some unused roles for packer

resource "google_service_account" "packer" {
  account_id = "negentropy-packer"
}

resource "google_project_iam_member" "packer-service-account-user" {
  project = local.google_project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.packer.email}"
}

resource "google_project_iam_member" "packer-storage-admin" {
  project = local.google_project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.packer.email}"
}

resource "google_project_iam_member" "packer-storage-object-admin" {
  project = local.google_project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.packer.email}"
}

resource "google_project_iam_member" "packer-compute-storage-admin" {
  project = local.google_project_id
  role    = "roles/compute.storageAdmin"
  member  = "serviceAccount:${google_service_account.packer.email}"
}
