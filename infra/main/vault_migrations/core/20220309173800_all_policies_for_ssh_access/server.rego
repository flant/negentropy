# rego for server role
# scope: project
# tenant_is_optional: false
# project_is_optional: false

# naming for package: negentropy.POLICY_NAME
package negentropy.server

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

project_is_passed {
    input.project_uuid
    input.project_uuid != ""
    }

server_is_passed {
    input.server_uuid
    input.server_uuid != ""
    }

# access status
default allow = false
allow {
    tenant_is_passed
    project_is_passed
    server_is_passed
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
} {
	err:="project is NOT passed"
    	not project_is_passed
        not show_paths
} {
	err:="server is NOT passed"
    	not server_is_passed
        not show_paths
}

tenant_uuid = t {
	t = input.tenant_uuid
    	not show_paths
} {
	t = "<tenant_uuid>"
    	show_paths
}

project_uuid = p {
	p = input.project_uuid
    	not show_paths
} {
	p = "<project_uuid>"
    	show_paths
}

server_uuid = t {
	t = input.server_uuid
    	not show_paths
} {
	t = "<server_uuid>"
    	show_paths
}

# rules for building vault policies
rules = [
	{"path":concat("/",["auth", "flant", "tenant", tenant_uuid, "project", project_uuid,"server", server_uuid, "posix_users"]),"capabilities":["read"]}] {
    allow
} {
    show_paths
}

ttl := requested_ttl {allow}

max_ttl := requested_max_ttl {allow}