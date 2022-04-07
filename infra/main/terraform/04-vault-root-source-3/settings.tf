terraform {
  required_providers {
    hashicorp-google = {
      version = "= 3.77.0"
      source  = "hashicorp/google"
    }
  }
  required_version = ">= 0.13"
  backend "gcs" {
    prefix = "vault-root-source-3"
  }
}

# terraform init -backend-config "bucket=$GOOGLE_PROJECT-terraform-state"

