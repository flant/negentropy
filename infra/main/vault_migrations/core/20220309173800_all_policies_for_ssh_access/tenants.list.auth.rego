# rego for tenants.list.auth
# scope: tenant
# tenant_is_optional: true

# naming for package: negentropy.POLICY_NAME
package negentropy.tenants.list.auth

default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

# show all possible vault policies
default show_paths=false
show_paths  {input.show_paths == true}

# access status
allow {
    not show_paths}

# rules for building vault policies (actually this path is allowed by default)
rules = [
	{"path":"auth/flant_iam_auth/tenant","capabilities":["list"]},
]

ttl := requested_ttl {allow}

max_ttl := requested_max_ttl {allow}