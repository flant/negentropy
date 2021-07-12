1. Create SA `terraform-bastion` and provide following IAM roles:
    ```
    Project Billing Manager
    Cloud KMS Admin
    Compute Admin
    Role Administrator
    Security Admin
    Service Account Admin
    Service Account User
    Network Management Admin
    Storage Admin
    Storage Object Admin
    ```
1. Create SA Key JSON and compact it with `jq -c`
1. Create file `env_vars`
    ```bash
    export GOOGLE_CREDENTIALS='<saKeyJSON>'
    export GOOGLE_PROJECT=negentropy-test-b
    ```
1. Create bucket for terraform-state `$GOOGLE_PROJECT-base-terraform-state`
   e.g. `negentropy-test-b-base-terraform-state` 
2. Create KMS and key with names from `apply.py` variables template:
   ```hcl
   gcp_ckms_seal_key_ring = "vault-test"
   gcp_ckms_seal_crypto_key = "vault-test-crypto-key"
   ```
   #### todo: closer to prod, different KMS's per vault (4)
3. Run following from `../terraform` directory: 
   ```shell
   source ../docker/env_vars
   terraform init -backend-config "bucket=$GOOGLE_PROJECT-base-terraform-state"
   terraform apply
   ```
   > Note: Enable following APIs in your GCP project:
   > - Cloud Resource Manager API
   > - Compute Engine API
   > - Identity and Access Management (IAM) API
   > 
   > Or during the first run terraform will give you a link to enable these APIs.
