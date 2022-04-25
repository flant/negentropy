package negentropy.ssh.open

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
       "valid_till": 999999999999
   },
   {
       "any_project": false,
       "need_approvals": 0,
       "options": {
           "max_ttl": "1200s",
           "ttl": "600s"
       },
       "projects": [
           "p1"
       ],
       "require_mfa": true,
       "rolebinding_uuid": "uuid2",
       "rolename": "query_servers",
       "tenant_uuid": "t1",
       "valid_till": 999999999999
   }
]
servers =[{"uuid":"0aaff1c0-0a93-4c15-9244-181aaeedd12d"},
           {"uuid":"s1"},
           {"uuid":"s2"},
           {"uuid":"s3"},
           {"uuid":"s4"}]
subject = {
               "uuid": "68e46dc0-b779-475d-b7a7-e93d548b04d5"
           }

ok_input_firts_rb = {
    "max_ttl": "200s",
    "project_uuid": "p1",
    "servers": [
        "0aaff1c0-0a93-4c15-9244-181aaeedd12d",
        "s1"
    ],
    "show_paths": false,
    "tenant_uuid": "t1",
    "ttl": "100s"
}

test_allow_by_first_rb_check_allow {
    allow
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_first_rb_check_errors {
    count(errors)==0
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_first_rb_check_rules {
    rules== [
    {
                "allowed_parameters": {
                    "principals": {
                        "2db561b02578945905f9688c540bc7489cf9dc7578d20b08cda636682c636a56",
                        "d56b1dfc8e81b509b007d0465f291524ccd4a5fb99f15eda5ecb6b57c47ba793"
                    }
                },
                "capabilities": [
                    "update"
                ],
                "path": "ssh/sign/signer",
                "required_parameters": [
                    "principals"
                ]
            }
    ]
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_first_rb_check_ttl {
     ttl=="100s"
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_first_rb_check_max_ttl {
     max_ttl=="200s"
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_first_rb_check_count_filtered_bindings {
     count(filtered_bindings)==2
     with input as ok_input_firts_rb
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

ok_input_second_rb_by_defult_ttl = {
    "project_uuid": "p1",
    "servers": [
        "0aaff1c0-0a93-4c15-9244-181aaeedd12d",
        "s1"
    ],
    "tenant_uuid": "t1",
}

test_allow_by_second_rb_check_allow {
    allow
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_second_rb_check_check_errors {
    count(errors)==0
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_second_rb_check_ttl {
     ttl=="600s"
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_second_rb_check_max_ttl {
     max_ttl=="1200s"
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_allow_by_second_rb_check_count_filtered_bindings {
     count(filtered_bindings)==1
     with input as ok_input_second_rb_by_defult_ttl
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

error_server_input = {
    "project_uuid": "p1",
    "servers": [
        "0aaff1c0-0a93-4c15-9244-181aaeedd12d",
        "server_error_uuid"
    ],
    "tenant_uuid": "t1",
}

test_forbid_by_wrong_server_check_forbid {
     not allow
     with input as error_server_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_wrong_server_check_errors {
     errors=={"servers are invalid: server_error_uuid"}
     with input as error_server_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_wrong_server_check_not_rules {
     not rules
     with input as error_server_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

error_ttl_input = {
    "ttl": "2000s",
    "project_uuid": "p1",
    "servers": [
        "0aaff1c0-0a93-4c15-9244-181aaeedd12d",
        "s1"
    ],
    "tenant_uuid": "t1",
}

test_forbid_by_wrong_ttl_forbid {
     not allow
     with input as error_ttl_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_wrong_ttl_check_errors {
     errors=={"no suitable rolebindings"}
     with input as error_ttl_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_wrong_ttl_check_not_rules {
     not rules
     with input as error_ttl_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

show_paths_input = {
    "show_paths":true
}

test_forbid_by_show_paths_check_not_allow {
    not allow
     with input as show_paths_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_show_paths_check_errors {
    count(errors)==0
     with input as show_paths_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_show_paths_check_rules {
    rules== [
    {
                "allowed_parameters": {
                    "principals": {
                        "sha256(server_uuid+user_uuud)",
                    }
                },
                "capabilities": [
                    "update"
                ],
                "path": "ssh/sign/signer",
                "required_parameters": [
                    "principals"
                ]
            }
    ]
     with input as show_paths_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_show_paths_check_not_ttl {
     not ttl
     with input as show_paths_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}

test_forbid_by_show_paths_check_not_max_ttl {
     not max_ttl
     with input as show_paths_input
     with data.effective_roles as effective_roles with data.servers as servers with data.subject as subject
}