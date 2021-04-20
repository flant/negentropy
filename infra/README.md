# Required roles
`gcp_builder_service_account` should have:
1. Compute Storage Admin - to create instance image
1. Service Account User - don't know for what
1. Storage Object Admin - to upload temporary image to S3 Storage

`vault-root-source` SA shoud have:
1. Logging Admin
1. Storage Admin

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
