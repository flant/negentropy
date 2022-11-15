terraform {
  required_providers {
    hashicorp-google = {
      version = "= 4.32.0"
      source  = "hashicorp/google"
    }
  }
  required_version = ">= 0.13"
  backend "gcs" {
    bucket = "negentropy-dev-terraform-state"
  }
}

provider "google" {
  project = local.project
}

resource "google_storage_bucket" "terraform-state" {
  name     = "negentropy-dev-terraform-state"
  location = local.location
  labels   = {}
}

resource "google_storage_bucket" "negentropy-dev-vault" {
  name     = "negentropy-dev-vault"
  location = local.location
  labels   = {}
}

resource "google_kms_key_ring" "main" {
  project = local.project
  name     = "vault"
  location = "europe"
}

# TODO: create own crypto key for each vault
resource "google_kms_crypto_key" "main" {
  name     = "vault-unseal"
  labels   = {}
  key_ring = google_kms_key_ring.main.id

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = "HSM"
  }
}