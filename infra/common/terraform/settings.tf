terraform {
  required_providers {
    hashicorp-google = {
      version = "= 3.65.0"
      source  = "hashicorp/google"
    }
  }
  required_version = ">= 0.13"
  backend "gcs" {
    prefix = "base"
  }
}

# 1. Create bucket for terraform-state `$GOOGLE_PROJECT-base-terraform-state`
# 2. Create KMS and key with names from `apply.py` variables template
# 3. terraform init -backend-config "bucket=$GOOGLE_PROJECT-base-terraform-state"
