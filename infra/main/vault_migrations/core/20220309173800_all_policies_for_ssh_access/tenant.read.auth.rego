# rego for tenant.read.auth role
# scope: tenant
# tenant_is_optional: false

# naming for package: negentropy.POLICY_NAME
package negentropy.tenant.read.auth

default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

# show all possible vault policies
default show_paths=false
show_paths  {input.show_paths == true}

filtered_bindings[r] {
	some i
	r := data.effective_roles[i]
        to_seconds_number(data.effective_roles[i].options.ttl)>=to_seconds_number(requested_ttl)
        to_seconds_number(data.effective_roles[i].options.max_ttl)>=to_seconds_number(requested_max_ttl)
}

rolebinding_exists {count(filtered_bindings) > 0}

tenant_is_passed  {
    input.tenant_uuid
    input.tenant_uuid != ""
    }

# access status
default allow = false
allow {
	rolebinding_exists
    not show_paths}

errors[err] {
	err:="no suitable rolebindings"
    	not rolebinding_exists
        not show_paths
} {
	err:="tenant is NOT passed"
		not tenant_is_passed
        not show_paths
}

tenant_uuid = t {
	t = input.tenant_uuid
    	not show_paths
} {
	t = "<tenant_uuid>"
    	show_paths
}

read_tenant_path = concat("/",["auth","flant_iam_auth","tenant",tenant_uuid]) {tenant_is_passed} {show_paths}
list_projects_path = concat("/",["auth","flant_iam_auth","tenant",tenant_uuid, "project"]) {tenant_is_passed} {show_paths}
read_project_path = concat("/",["auth","flant_iam_auth","tenant",tenant_uuid, "project", "+"]) {tenant_is_passed} {show_paths}

# rules for building vault policies
rules = [
	{"path":read_tenant_path,"capabilities":["read"]},
    {"path":list_projects_path,"capabilities":["list"]},
    {"path":read_project_path,"capabilities":["read"]},
] {
    allow
} {
    show_paths
}

ttl := requested_ttl {allow}

max_ttl := requested_max_ttl {allow}

# convert to seconds
to_seconds_number(t) = x {
	x=to_number(t)
}{
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
	 x = value ; endswith(lower_t, "s")
}{
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
	 x = value*60 ; endswith(lower_t, "m")
}{
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
     x = value*3600 ; endswith(lower_t, "h")
}
