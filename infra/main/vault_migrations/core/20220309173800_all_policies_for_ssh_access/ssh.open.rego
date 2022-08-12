# rego for ssh.open role
# scope: project
# tenant_is_optional: false
# project_is_optional: false

# naming for package: negentropy.POLICY_NAME
package negentropy.ssh.open

default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

filtered_bindings[r] {
	some i
	r := data.effective_roles[i]
        to_seconds_number(data.effective_roles[i].options.ttl)>=to_seconds_number(requested_ttl)
        to_seconds_number(data.effective_roles[i].options.max_ttl)>=to_seconds_number(requested_max_ttl)
}

rolebinding_exists {count(filtered_bindings) > 0}


valid_servers_uuid [server_uuid] {
	some i
    server_uuid := input.servers[i]
     	server_uuid == data.servers[_].uuid
}

input_servers [server_uuid] {
	some i
    server_uuid := input.servers[i]
}

invalid_servers = input_servers - valid_servers_uuid

all_servers_ok {count(invalid_servers)==0}

tenant_is_passed  {
    input.tenant_uuid
    input.tenant_uuid != ""
    }

project_is_passed {
    input.project_uuid
    input.project_uuid != ""
    }

# show all possible vault policies
default show_paths=false
show_paths  {input.show_paths == true}

# access status
default allow = false
allow {
	rolebinding_exists
    all_servers_ok
    tenant_is_passed
    project_is_passed
	not show_paths
    }

errors[err] {
	err:="no suitable rolebindings"
    	not rolebinding_exists
        not show_paths
} {
	err:=concat(": ", ["servers are invalid", concat(", ", invalid_servers)])
    	not all_servers_ok
        not show_paths
} {
	err:="tenant_uuid not passed"
    	not tenant_is_passed
        not show_paths
} {
	err:="project_uuid not passed"
    	not project_is_passed
        not show_paths
}

principals[principal] {
		some i
 	       principal := crypto.sha256(concat("",[input.servers[i], data.subject.uuid]))
           not show_paths
}{
	principal := "sha256(server_uuid1+user_uuud),sha256(server_uuid2+user_uuud)"
    	show_paths
}

valid_principals = concat(",", sort(principals))

# rules for building vault policies
rules = [
	{
    	"path":"ssh/sign/signer",
    	"capabilities":["update"],
	    "required_parameters":["valid_principals"],
        "allowed_parameters":
        {
        	"valid_principals":[valid_principals],
            "*":[]
        }
    }
    ]	{allow}
    	{show_paths}

ttl := requested_ttl {allow}

max_ttl := requested_max_ttl {allow}

# cvonvert to seconds
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

