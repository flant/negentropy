# rego for servers.query role
# scope: project
# tenant_is_optional: true
# project_is_optional: true

# naming for package: negentropy.POLICY_NAME
package negentropy.servers.query

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

project_is_passed {
    input.project_uuid
    input.project_uuid != ""
    }

project_is_passed_tenant_is_not {
	project_is_passed
    not tenant_is_passed
}


# access status
default allow = false
allow {
	rolebinding_exists
    not project_is_passed_tenant_is_not
    not show_paths}

errors[err] {
	err:="no suitable rolebindings"
    	not rolebinding_exists
        not show_paths
} {
	err:="tenant is NOT passed, but project is"
    	project_is_passed_tenant_is_not
        not show_paths
}

project_uuid = p {
	p = input.project_uuid
    	not show_paths
} {
	p = "<project_uuid>"
    	show_paths
}

tenant_uuid = t {
	t = input.tenant_uuid
    	not show_paths
} {
	t = "<tenant_uuid>"
    	show_paths
}

global_path = "auth/flant_iam_auth/query_server" {not tenant_is_passed; not project_is_passed} {show_paths}
with_tenant_path = concat("/",["auth","flant_iam_auth","tenant",tenant_uuid,"query_server"]) {not project_is_passed; tenant_is_passed} {show_paths}
with_project_path = concat("/",["auth","flant_iam_auth","tenant",tenant_uuid,"project",project_uuid,"query_server"]) {project_is_passed; tenant_is_passed} {show_paths}

# fill rules
pre_rules  [rule]{
	rule={"path":global_path,"capabilities":["read"]}
} {
	rule={"path":with_tenant_path,"capabilities":["read"]}
} {
	rule={"path":with_project_path,"capabilities":["read"]}
}

# rules for building vault policies
rules = pre_rules {
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

# full response for ok_input_firts_rb
# -----------------------------------


#{
#    "allow": true,
#    "errors": [],
#    "filtered_bindings": [
#        {
#            "any_project": false,
#            "need_approvals": 0,
#            "options": {
#                "max_ttl": "1600s",
#                "ttl": "800s"
#            },
#            "projects": [
#                "p1"
#            ],
#            "require_mfa": true,
#            "rolebinding_uuid": "uuid2",
#            "rolename": "query_servers",
#            "tenant_uuid": "t1",
#            "valid_till": 0
#        },
#        {
#            "any_project": false,
#            "need_approvals": 0,
#            "options": {
#                "max_ttl": "200s",
#                "ttl": "100s"
#            },
#            "projects": [
#                "p1"
#            ],
#            "require_mfa": false,
#            "rolebinding_uuid": "uuid1",
#            "rolename": "query_servers",
#            "tenant_uuid": "t1",
#            "valid_till": 0
#        }
#    ],
#    "max_ttl": "200s",
#    "pre_rules": [
#        {
#            "capabilities": [
#                "read"
#            ],
#            "path": "auth/flant/tenant/t1/project/p1/query_server"
#        }
#    ],
#    "project_is_passed": true,
#    "project_uuid": "p1",
#    "requested_max_ttl": "200s",
#    "requested_ttl": "100s",
#    "rolebinding_exists": true,
#    "rules": [
#        {
#            "capabilities": [
#                "read"
#            ],
#            "path": "auth/flant/tenant/t1/project/p1/query_server"
#        }
#    ],
#    "show_paths": false,
#    "tenant_is_passed": true,
#    "tenant_uuid": "t1",
#    "ttl": "100s",
#    "with_project_path": "auth/flant/tenant/t1/project/p1/query_server"
#}