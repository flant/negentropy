terraform {
  required_providers {
    hashicorp-google = {
      version = "= 3.65.0"
      source  = "hashicorp/google"
    }
  }
  required_version = ">= 0.13"
  backend "gcs" {
    # bucket = "negentropy-terraform-state"
    prefix = "vault-configurator"
  }
}
