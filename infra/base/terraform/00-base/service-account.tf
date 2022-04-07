# TODO: migrate to individual service accounts for each vault and kafka instances
# TODO: cleanup roles, maybe there are some unused roles for packer and terraform

resource "google_service_account" "negentropy-base" {
  account_id   = "negentropy-base"
}

resource "google_project_iam_member" "negentropy-storage-admin" {
  project = local.google_project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}

resource "google_project_iam_member" "negentropy-storage-object-admin" {
  project = data.google_project.project.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}

resource "google_project_iam_member" "negentropy-compute-storage-admin" {
  project = local.google_project_id
  role    = "roles/compute.storageAdmin"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}

resource "google_project_iam_member" "negentropy-compute-admin" {
  project = data.google_project.project.project_id
  role    = "roles/compute.admin"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}

resource "google_project_iam_member" "negentropy-kms-admin" {
  project = data.google_project.project.project_id
  role    = "roles/cloudkms.admin"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}

resource "google_project_iam_member" "negentropy-kms-crypto-key-encrypter-decrypter" {
  project = data.google_project.project.project_id
  role    = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}

resource "google_project_iam_member" "negentropy-cloud-dns" {
  project = data.google_project.project.project_id
  role    = "roles/dns.admin"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}

resource "google_project_iam_member" "negentropy-privateca-certificate-manager" {
  project = data.google_project.project.project_id
  role    = "roles/privateca.certificateManager"
  member  = "serviceAccount:${google_service_account.negentropy-base.email}"
}
