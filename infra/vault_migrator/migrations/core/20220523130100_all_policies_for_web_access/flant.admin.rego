# rego for flant.admin role
# scope: tenant
# tenant_is_optional: true

# naming for package: negentropy.POLICY_NAME
package negentropy.flant.admin

default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

# show all possible vault policies
default show_paths=false
show_paths  {input.show_paths == true}

rolebinding_exists {count(data.effective_roles) > 0}

# access status
default allow = false
allow {
	rolebinding_exists
    not show_paths}

errors[err] {
	err:="no suitable rolebindings"
    	not rolebinding_exists
        not show_paths
}

# rules for building vault policies
rules = [
	{"path":"flant/client","capabilities":["create", "update"]},
	{"path":"flant/client/","capabilities":["read"]},
    {"path":"flant/client/+","capabilities":["read", "update", "delete"]},
    {"path":"flant/team","capabilities":["create", "update"]},
	{"path":"flant/team/","capabilities":["read"]},
    {"path":"flant/team/+","capabilities":["read", "update", "delete"]},
    {"path":"flant/team/+/teammate","capabilities":["create", "update"]},
    {"path":"flant/teammate/","capabilities":["read"]},
    {"path":"flant/team/+/teammate/+","capabilities":["read", "update", "delete"]}
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