resource "google_service_account" "terraform" {
  account_id   = "terraform-runner"
  display_name = "A service account for run terraform apply on vault-conf and vault-conf-conf instances"
}

resource "google_project_iam_member" "terraform-service-account-user" {
  project = data.google_project.project.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

resource "google_project_iam_member" "terraform-service-account-admin" {
  project = data.google_project.project.project_id
  role    = "roles/iam.serviceAccountAdmin"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

resource "google_project_iam_member" "terraform-security-admin" {
  project = data.google_project.project.project_id
  role    = "roles/iam.securityAdmin"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

resource "google_project_iam_member" "terraform-compute-admin" {
  project = data.google_project.project.project_id
  role    = "roles/compute.admin"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

resource "google_project_iam_member" "terraform-storage-admin" {
  project = data.google_project.project.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

resource "google_project_iam_member" "terraform-storage-object-admin" {
  project = data.google_project.project.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

resource "google_project_iam_member" "terraform-kms-crypto-key-encrypter-decrypter" {
  project = data.google_project.project.project_id
  role    = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

resource "google_project_iam_member" "terraform-kms-admin" {
  project = data.google_project.project.project_id
  role    = "roles/cloudkms.admin"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}

# TODO: remove "provider = google-beta"
resource "google_project_iam_member" "terraform-privateca-certmanager" {
  provider = google-beta
  project = data.google_project.project.project_id
  role    = "roles/privateca.certificateManager"
  member  = "serviceAccount:${google_service_account.terraform.email}"
}
