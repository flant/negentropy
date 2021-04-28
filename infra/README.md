# Required roles
SA specified in variable `gcp_builder_service_account` should have:
1. Compute Storage Admin - to create instance image
1. Service Account User - don't know for what
1. Storage Object Admin - to upload temporary image to S3 Storage

All main and configurator instances SA should have:
1. Logging Admin - to push logs to Cloud Logs

`vault-conf` instance SA should have:
1. Storage Object Admin - to access bucket `gcp_vault_conf_bucket`

`vault-conf-conf` instance SA should have:
1. Storage Object Admin - to access bucket `gcp_vault_conf_conf_bucket`

`vault-root-source` instance SA should have:
1. Storage Object Admin - to access bucket `gcp_vault_root_source_bucket`

`vault-auth` instance SA should have:
1. Storage Object Admin - to access buckets `$(hostname).gcp_vault_auth_bucket_trailer`

`kafka` instance SA should have:
1. Storage Object Admin - to access bucket `kafka_bucket`
1. CA Service Operation Manager - to download CA
1. CA Service Certificate Manager - to create new certificate

# Quickstart
Create `variables.pkrvars.hcl` referencing to `variables.pkrvars.hcl.example`.

# Building base image in MacOS
To allow qemu use hfv accelerator we need to sign qemu binary with following `entitlements.xml`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.hypervisor</key>
    <true/>
</dict>
</plist>
```
Then run:
```bash
codesign -s - --entitlements entitlements.xml --force /usr/local/bin/qemu-system-x86_64
```
## `variables.pkrvars.hcl`
Define `hvf` accelerator:
```hcl
accel = "hvf"
```
Optionally set display options:
```hcl
display = "cocoa"
headless = "false"
```
