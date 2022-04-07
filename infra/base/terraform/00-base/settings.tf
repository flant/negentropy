terraform {
  required_providers {
    hashicorp-google = {
      version = "= 4.6.0"
      source  = "hashicorp/google"
    }
  }
  required_version = ">= 0.13"
  backend "gcs" {
    prefix = "base"
  }
}

# terraform init -backend-config "bucket=$GOOGLE_PROJECT-base-terraform-state"
