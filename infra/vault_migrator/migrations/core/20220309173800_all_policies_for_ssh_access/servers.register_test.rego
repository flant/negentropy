package negentropy.servers.register

# example of data
effective_roles = [
   {
       "any_project": false,
       "need_approvals": 0,
       "options": {
           "max_ttl": "20s",
           "ttl": "10s"
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
           "max_ttl": "160s",
           "ttl": "80s"
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
    "max_ttl": "20s",
    "project_uuid": "p1",
    "show_paths": false,
    "tenant_uuid": "t1",
    "ttl": "10s"
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
    # here we got array
    rules == [
                     {
                         "capabilities": [
                             "create",
                             "update"
                         ],
                         "path": "flant/tenant/t1/project/p1/register_server*"
                     },
                     {
                         "capabilities": [
                             "create",
                             "read",
                             "update",
                             "delete",
                             "list"
                         ],
                         "path": "flant/tenant/t1/project/p1/server*"
                     }
                 ]
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_ttl {
     ttl=="10s"
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_max_ttl {
     max_ttl=="20s"
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

test_allow_by_first_rb_check_count_filtered_bindings {
     count(filtered_bindings)==2
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles
}

ok_input_second_rb_by_defult_ttl = {
    "project_uuid": "p1",
    "tenant_uuid": "t1",
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
     ttl=="60s"
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_second_rb_check_max_ttl {
     max_ttl=="120s"
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_second_rb_check_count_filtered_bindings {
     count(filtered_bindings)==1
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles
}

error_not_passed_project = {
    "tenant_uuid": "t1",
}

test_forbid_by_not_passed_project_check_forbid {
     not allow
     with input as error_not_passed_project
     with data.effective_roles as effective_roles
}

test_forbid_by_not_passed_project_check_errors {
     errors=={"project is NOT passed"}
     with input as error_not_passed_project
     with data.effective_roles as effective_roles
}

test_forbid_by_not_passed_project_check_not_rules {
     not rules
     with input as error_not_passed_project
     with data.effective_roles as effective_roles
}

error_ttl_input = {
    "ttl": "2000s",
    "project_uuid": "p1",
    "tenant_uuid": "t1",
}

test_forbid_by_wrong_ttl_forbid {
     not allow
     with input as error_ttl_input
     with data.effective_roles as effective_roles
}

test_forbid_by_wrong_ttl_check_errors {
     errors=={"no suitable rolebindings"}
     with input as error_ttl_input
     with data.effective_roles as effective_roles
}

test_forbid_by_wrong_ttl_check_not_rules {
     not rules
     with input as error_ttl_input
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
    # we got here array
    rules== [
                    {
                        "capabilities": [
                            "create",
                            "update"
                        ],
                        "path": "flant/tenant/<tenant_uuid>/project/<project_uuid>/register_server*"
                    },
                    {
                        "capabilities": [
                            "create",
                            "read",
                            "update",
                            "delete",
                            "list"
                        ],
                        "path": "flant/tenant/<tenant_uuid>/project/<project_uuid>/server*"
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
#    "max_ttl": "20s",
#    "project_is_passed": true,
#    "project_uuid": "p1",
#    "requested_max_ttl": "20s",
#    "requested_ttl": "10s",
#    "rolebinding_exists": true,
#    "rules": [
#        {
#            "capabilities": [
#                "create",
#                "update"
#            ],
#            "path": "flant/tenant/t1/project/p1/register_server*"
#        },
#        {
#            "capabilities": [
#                "create",
#                "read",
#                "update",
#                "delete",
#                "list"
#            ],
#            "path": "flant/tenant/t1/project/p1/server*"
#        }
#    ],
#    "show_paths": false,
#    "tenant_is_passed": true,
#    "tenant_uuid": "t1",
#    "ttl": "10s"
#}