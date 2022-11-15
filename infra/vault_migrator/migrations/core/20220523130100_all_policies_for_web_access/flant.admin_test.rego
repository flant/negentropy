package negentropy.flant.admin

# example of data
effective_roles = [
   {
       "any_project": false,
       "need_approvals": 0,
       "projects": [],
       "require_mfa": false,
       "rolebinding_uuid": "uuid1",
       "rolename": "flant.teammate",
       "tenant_uuid": "",
       "valid_till": 0
   },
   {
       "any_project": false,
       "need_approvals": 0,
       "projects": [],
       "require_mfa": true,
       "rolebinding_uuid": "uuid2",
       "rolename": "flant.teammate",
       "tenant_uuid": "",
       "valid_till": 0
   }
]

ok_input_one = {
    "max_ttl": "200s",
    "show_paths": false,
    "tenant_uuid": "",
    "ttl": "100s"
}

test_allow_one_check_allow {
    allow
     with input as ok_input_one
     with data.effective_roles as effective_roles
}

test_allow_one_check_count_filtered_bindings {
    count(filtered_bindings)==2
     with input as ok_input_one
     with data.effective_roles as effective_roles
}

test_allow_by_input_one_check_errors {
    count(errors)==0
     with input as ok_input_one
     with data.effective_roles as effective_roles
}

test_allow_by_input_one_check_rules {
    # we got array here
    rules == [
        {
            "capabilities": [
                "create",
                "update"
            ],
            "path": "flant/client"
        },
        {
            "capabilities": [
                "read"
            ],
            "path": "flant/client/"
        },
        {
            "capabilities": [
                "read",
                "update",
                "delete"
            ],
            "path": "flant/client/+"
        },
        {
            "capabilities": [
                "create",
                "update"
            ],
            "path": "flant/team"
        },
        {
            "capabilities": [
                "read"
            ],
            "path": "flant/team/"
        },
        {
            "capabilities": [
                "read",
                "update",
                "delete"
            ],
            "path": "flant/team/+"
        },
        {
            "capabilities": [
                "create",
                "update"
            ],
            "path": "flant/team/+/teammate"
        },
        {
            "capabilities": [
                "read"
            ],
            "path": "flant/teammate/"
        },
        {
            "capabilities": [
                "read",
                "update",
                "delete"
            ],
            "path": "flant/team/+/teammate/+"
        }
    ]
     with input as ok_input_one
     with data.effective_roles as effective_roles
}

test_allow_by_input_one_check_ttl {
     ttl=="100s"
     with input as ok_input_one
     with data.effective_roles as effective_roles
}

test_allow_by_input_one_check_max_ttl {
     max_ttl=="200s"
     with input as ok_input_one
     with data.effective_roles as effective_roles
}

ok_input_default_ttl = {}

test_allow_by_input_default_check_allow {
    allow
     with input as ok_input_default_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_input_default_check_check_errors {
    count(errors)==0
     with input as ok_input_default_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_input_default_check_ttl {
     ttl=="600s"
     with input as ok_input_default_ttl
     with data.effective_roles as effective_roles
}

test_allow_by_input_default_check_max_ttl {
     max_ttl=="1200s"
     with input as ok_input_default_ttl
     with data.effective_roles as effective_roles
}

empty_effective_roles = []

test_forbid_by_absense_rolebindings_forbid {
     not allow
     with input as ok_input_default_ttl
     with data.effective_roles as empty_effective_roles
}

test_forbid_by_absense_rolebindings_check_not_filtered_bindings {
     not filtered_bindings
     with input as ok_input_default_ttl
     with data.effective_roles as empty_effective_roles
}

test_forbid_by_absense_rolebindings_check_errors {
     errors=={"no suitable rolebindings"}
     with input as ok_input_default_ttl
     with data.effective_roles as empty_effective_roles
}

test_forbid_by_absense_rolebindings_check_not_rules {
     not rules
     with input as ok_input_default_ttl
     with data.effective_roles as empty_effective_roles
}

test_forbid_by_absense_rolebindings_check_not_ttl {
     not ttl
     with input as ok_input_default_ttl
     with data.effective_roles as empty_effective_roles
}

test_forbid_by_absense_rolebindings_check_not_max_ttl {
     not max_ttl
     with input as ok_input_default_ttl
     with data.effective_roles as empty_effective_roles
}

show_paths_input = {
    "show_paths":true
}

test_forbid_by_show_paths_check_not_allow {
     not allow
     with input as show_paths_input
     with data.effective_roles as effective_roles
}

test_forbid_by_show_paths_check_not_filtered_bindings {
     not filtered_bindings
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
    rules == [
        {
            "capabilities": [
                "create",
                "update"
            ],
            "path": "flant/client"
        },
        {
            "capabilities": [
                "read"
            ],
            "path": "flant/client/"
        },
        {
            "capabilities": [
                "read",
                "update",
                "delete"
            ],
            "path": "flant/client/+"
        },
        {
            "capabilities": [
                "create",
                "update"
            ],
            "path": "flant/team"
        },
        {
            "capabilities": [
                "read"
            ],
            "path": "flant/team/"
        },
        {
            "capabilities": [
                "read",
                "update",
                "delete"
            ],
            "path": "flant/team/+"
        },
        {
            "capabilities": [
                "create",
                "update"
            ],
            "path": "flant/team/+/teammate"
        },
        {
            "capabilities": [
                "read"
            ],
            "path": "flant/teammate/"
        },
        {
            "capabilities": [
                "read",
                "update",
                "delete"
            ],
            "path": "flant/team/+/teammate/+"
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