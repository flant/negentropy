# rego for tenant.read role
# scope: tenant
# tenant_is_optional: false

# naming for package: negentropy.POLICY_NAME
package negentropy.tenant.read

default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

# show all possible vault policies
default show_paths=false
show_paths  {input.show_paths == true}

rolebinding_exists {count(data.effective_roles) > 0}

tenant_is_passed  {
    input.tenant_uuid
    input.tenant_uuid != ""
    }

# access status
default allow = false
allow {
	rolebinding_exists
    tenant_is_passed
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

# rules for building vault policies
rules = [
	# client/tenant:
	{"path":concat("/",["flant", "client", tenant_uuid]), #  read client
	"capabilities":["read"]},
	{"path":concat("/",["flant", "tenant", tenant_uuid]), #  read tenant
	"capabilities":["read"]},

    # flow project:
	{"path":concat("/",["flant", "client", tenant_uuid, "project", "+"]), #  read project
	"capabilities":["read"]},
    {"path":concat("/",["flant", "client", tenant_uuid, "project/"]), #  list project
	"capabilities":["read"]},

	# iam project:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "project", "+"]), #  read project
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "project/"]), #  list project
	"capabilities":["read"]},

	# iam group:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "group", "+"]), #  read group
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "group/"]), #  list group
	"capabilities":["read"]},

	# iam identity_sharing:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "identity_sharing", "+"]), #  read identity_sharing
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "identity_sharing/"]), #  list identity_sharing
	"capabilities":["read"]},

	# iam role_binding:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "role_binding", "+"]), #  read role_binding
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "role_binding/"]), #  list role_binding
	"capabilities":["read"]},

	# iam service_account:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "service_account", "+"]), #  read service_account
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "service_account/"]), #  list service_account
	"capabilities":["read"]},

	# iam service_account password:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "service_account",  "+", "password", "+"]), #  read service_account  password
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "service_account",  "+", "password/"]), #  list service_account password
	"capabilities":["read"]},

	# iam service_account multipass:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "service_account",  "+", "multipass", "+"]), #  read service_account  multipass
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "service_account",  "+", "multipass/"]), #  list service_account multipass
	"capabilities":["read"]},

	# iam user:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "user", "+"]), #  read user
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "user/"]), #  list user
	"capabilities":["read"]},

	# iam user multipass:
	{"path":concat("/",["flant", "tenant", tenant_uuid, "user",  "+", "multipass", "+"]), #  read user  multipass
	"capabilities":["read"]},
    {"path":concat("/",["flant", "tenant", tenant_uuid, "user",  "+", "multipass/"]), #  list user multipass
	"capabilities":["read"]}
] {
    allow
} {
    show_paths
}

ttl := requested_ttl {
	allow
    to_seconds_number(requested_ttl) <= 600
    }

max_ttl := requested_max_ttl {
	allow
    to_seconds_number(requested_max_ttl) <= 1200
    }

filtered_bindings := data.effective_roles {allow}

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