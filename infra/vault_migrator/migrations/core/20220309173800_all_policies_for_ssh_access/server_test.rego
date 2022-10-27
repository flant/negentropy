package negentropy.server

# example of data
effective_roles = [
   {
       "any_project": false,
       "need_approvals": 0,
       "options": {
           "max_ttl": "200s",
           "ttl": "100s"
       },
       "projects": [
           "p1"
       ],
       "require_mfa": false,
       "rolebinding_uuid": "uuid1",
       "rolename": "query_servers",
       "tenant_uuid": "t1",
       "valid_till": 0
   },
   {
       "any_project": false,
       "need_approvals": 0,
       "options": {
           "max_ttl": "1600s",
           "ttl": "800s"
       },
       "projects": [
           "p1"
       ],
       "require_mfa": true,
       "rolebinding_uuid": "uuid2",
       "rolename": "query_servers",
       "tenant_uuid": "t1",
       "valid_till": 0
   }
]

ok_input_firts_rb = {
    "max_ttl": "200s",
    "show_paths": false,
    "tenant_uuid": "t1",
    "project_uuid": "p1",
    "server_uuid": "s1",
    "ttl": "100s"
}

test_allow_by_first_rb_check_allow {
    allow
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_errors {
    count(errors)==0
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_rules {
    # here we got array, not the set
    rules ==[
                    {
                        "capabilities": [
                            "read",
                        ],
                        "path": "auth/flant/tenant/t1/project/p1/server/s1/posix_users"
                    }
                ]
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_ttl {
     ttl=="100s"
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_max_ttl {
     max_ttl=="200s"
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_count_filtered_bindings {
     rolebinding_exists
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

ok_input_second_rb_by_defult_ttl = {
    "project_uuid": "p1",
    "tenant_uuid": "t1",
    "server_uuid": "s1"
}

test_allow_by_second_rb_check_allow {
    allow
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_second_rb_check_check_errors {
    count(errors)==0
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_second_rb_check_ttl {
     ttl=="600s"
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_second_rb_check_max_ttl {
     max_ttl=="1200s"
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_second_rb_check_rolebinding_exists {
     rolebinding_exists
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

error_not_passed_server = {
    "tenant_uuid": "t1",
    "project_uuid": "p1",
}

test_forbid_by_not_passed_server {
     not allow
     with input as error_not_passed_server
     with data.effective_roles as effective_roles
}

test_forbid_by_not_passed_server {
     errors=={"server is NOT passed"}
     with input as error_not_passed_server
     with data.effective_roles as effective_roles
}

test_forbid_by_not_passed_server {
     not rules
     with input as error_not_passed_server
     with data.effective_roles as effective_roles
}

show_paths_input = {
    "show_paths":true
}

test_forbid_by_show_paths_check_not_allow {
     not allow
     with input as show_paths_input
     with data.effective_roles as effective_roles
}

test_forbid_by_show_paths_check_errors {
    count(errors)==0
     with input as show_paths_input
     with data.effective_roles as effective_roles
}

test_forbid_by_show_paths_check_rules {
    # we got array here
    rules== [
                    {
                        "capabilities": [
                            "read"
                        ],
                        "path": "auth/flant/tenant/<tenant_uuid>/project/<project_uuid>/server/<server_uuid>/posix_users"
                    }
                ]
     with input as show_paths_input
     with data.effective_roles as effective_roles
}

test_forbid_by_show_paths_check_not_ttl {
     not ttl
     with input as show_paths_input
     with data.effective_roles as effective_roles
}

test_forbid_by_show_paths_check_not_max_ttl {
     not max_ttl
     with input as show_paths_input
     with data.effective_roles as effective_roles
}

# full response for ok_input_firts_rb
# -----------------------------------

#{
#    "allow": true,
#    "errors": [],
#    "max_ttl": "200s",
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
#            "path": "auth/flant/tenant/t1/project/p1/server/s1/posix_users"
#        }
#    ],
#    "server_is_passed": true,
#    "server_uuid": "s1",
#    "show_paths": false,
#    "tenant_is_passed": true,
#    "tenant_uuid": "t1",
#    "ttl": "100s"
#}