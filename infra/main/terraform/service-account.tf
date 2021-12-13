# TODO: naming refactoring is needed

resource "google_service_account" "vault_storage_and_kms_access" {
  account_id   = "vault-storage-and-kms-access"
  display_name = "A service account for dedicated vault instances"
}

resource "google_project_iam_member" "vault-storage-object-admin" {
  project = data.google_project.project.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.vault_storage_and_kms_access.email}"
}

resource "google_project_iam_member" "vault-kms-crypto-key-encrypter-decrypter" {
  project = data.google_project.project.project_id
  role    = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member  = "serviceAccount:${google_service_account.vault_storage_and_kms_access.email}"
}

resource "google_project_iam_member" "vault-cloud-dns" {
  project = data.google_project.project.project_id
  role    = "roles/dns.admin"
  member  = "serviceAccount:${google_service_account.vault_storage_and_kms_access.email}"
}

# for kafka
resource "google_project_iam_member" "kafka-privateca-certificate-manager" {
  project = data.google_project.project.project_id
  role    = "roles/privateca.certificateManager"
  member  = "serviceAccount:${google_service_account.vault_storage_and_kms_access.email}"
}
