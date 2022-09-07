#GAR
resource "google_artifact_registry_repository" "registry" {
  location      = local.region
  repository_id = local.project
  format        = "DOCKER"
}

#Users
resource "google_service_account" "gar_github_runner" {
  account_id   = "gar-github-runner"
  project = local.project
  display_name = "GAR SA User for Github Runner "
}

resource "google_artifact_registry_repository_iam_member" "gar_github_runner" {
  repository = google_artifact_registry_repository.registry.name
  project = local.project
  location = google_artifact_registry_repository.registry.location
  role = "roles/artifactregistry.repoAdmin"
  member = "serviceAccount:${google_service_account.gar_github_runner.email}"
}

output "gar_github_runner_service_account_email" {
  description = "Service account email (for single use)."
  value       = google_service_account.gar_github_runner.email
}

resource "google_service_account" "gar_ro_user" {
  account_id   = "gar-ro-user"
  project = local.project
  display_name = "GAR SA Read-only User (RegistrySecret)"
}

resource "google_artifact_registry_repository_iam_member" "gar_ro_user" {
  repository = google_artifact_registry_repository.registry.name
  project = local.project
  location = google_artifact_registry_repository.registry.location
  role = "roles/artifactregistry.reader"
  member = "serviceAccount:${google_service_account.gar_ro_user.email}"
}

output "gar_ro_user_service_account_email" {
  description = "Service account email (for single use)."
  value       = google_service_account.gar_ro_user.email
}

resource "google_service_account" "gar_rw_user" {
  account_id   = "gar-rw-user"
  project = local.project
  display_name = "GAR SA Read-write User"
}

resource "google_artifact_registry_repository_iam_member" "gar_rw_user" {
  repository = google_artifact_registry_repository.registry.name
  project = local.project
  location = google_artifact_registry_repository.registry.location
  role = "roles/artifactregistry.repoAdmin"
  member = "serviceAccount:${google_service_account.gar_rw_user.email}"
}

output "gar_rw_user_service_account_email" {
  description = "Service account email (for single use)"
  value       = google_service_account.gar_rw_user.email
}
