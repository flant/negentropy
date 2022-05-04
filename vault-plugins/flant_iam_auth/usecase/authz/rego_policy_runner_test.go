package authz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
)

var (
	uuid1          = "uuid1"
	uuid2          = "uuid2"
	uuid3          = "uuid3"
	uuid4          = "uuid4"
	effectiveRoles = []iam_usecase.EffectiveRole{
		{ // not need MFA or approvals
			RoleName:        "ssh.open",
			RoleBindingUUID: uuid1,
			TenantUUID:      "t1",
			ValidTill:       999999999999,
			RequireMFA:      false,
			AnyProject:      false,
			Projects:        []string{"p1"},
			NeedApprovals:   0,
			Options:         map[string]interface{}{"ttl": "100s", "max_ttl": "200s"},
		},
		{ // need MFA, not need approvals
			RoleName:        "ssh.open",
			RoleBindingUUID: uuid2,
			TenantUUID:      "t1",
			ValidTill:       999999999999,
			RequireMFA:      true,
			AnyProject:      false,
			Projects:        []string{"p1"},
			NeedApprovals:   0,
			Options:         map[string]interface{}{"ttl": "200s", "max_ttl": "400s"},
		},
		{ // not need MFA,  need 1 approval
			RoleName:        "ssh.open",
			RoleBindingUUID: uuid3,
			TenantUUID:      "t1",
			ValidTill:       999999999999,
			RequireMFA:      true,
			AnyProject:      false,
			Projects:        []string{"p1"},
			NeedApprovals:   1,
			Options:         map[string]interface{}{"ttl": "400s", "max_ttl": "800s"},
		},
		{ // not need MFA,  need 2 approvals
			RoleName:        "ssh.open",
			RoleBindingUUID: uuid4,
			TenantUUID:      "t1",
			ValidTill:       999999999999,
			RequireMFA:      true,
			AnyProject:      false,
			Projects:        []string{"p1"},
			NeedApprovals:   2,
			Options:         map[string]interface{}{"ttl": "800s", "max_ttl": "1600s"},
		},
	}

	regoPolicy = `
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

tenant_is_passed  {input.tenant_uuid}
project_is_passed {input.project_uuid}

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
	err:=concat(": ",["servers are invalid", concat(",", invalid_servers)])
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
	principal := "sha256(server_uuid+user_uuud)"
    	show_paths
}

# rules for building vault policies
rules = [
	{
    	"path":"ssh/sign/signer",
    	"capabilities":["update"],
	    "required_parameters":["valid_principals"],
        "allowed_parameters":
        {
        	"valid_principals":principals,
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
}`

	sshPolicy = model.Policy{
		Name: "ssh.open",
		Rego: regoPolicy,
	}

	sshVaultRules = []Rule{{Path: "ssh/sign/signer", AllowedParameters: map[string][]string{
		"valid_principals": {"2db561b02578945905f9688c540bc7489cf9dc7578d20b08cda636682c636a56", "d56b1dfc8e81b509b007d0465f291524ccd4a5fb99f15eda5ecb6b57c47ba793"},
		"*":                {},
	}, RequiredParameters: []string{"valid_principals"}, Update: true}}

	subject = model.Subject{
		Type:       "user",
		UUID:       "68e46dc0-b779-475d-b7a7-e93d548b04d5",
		TenantUUID: "t1",
	}

	serverExtData = map[string]interface{}{
		"servers": []ext.Server{
			{UUID: "0aaff1c0-0a93-4c15-9244-181aaeedd12d"},
			{UUID: "s1"},
			{UUID: "s2"},
		},
	}
)

func Claims(ttl, maxTTL string) map[string]interface{} {
	return map[string]interface{}{
		// "role":         "ssh.open",
		"tenant_uuid":  "t1",
		"project_uuid": "p1",
		"ttl":          ttl,
		"max_ttl":      maxTTL,
		"servers": []string{
			"0aaff1c0-0a93-4c15-9244-181aaeedd12d",
			"s1",
		},
	}
}

func Test_NotAllowedDueToBigTTL(t *testing.T) {
	claims := Claims("10000s", "100s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, subject, serverExtData, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Allow)
	require.Equal(t, []string{"no suitable rolebindings"}, result.Errors)
	require.Nil(t, result.BestEffectiveRole)
	require.Nil(t, result.VaultRules)
}

func Test_NotAllowedDueToBigMaxTTL(t *testing.T) {
	claims := Claims("100s", "10000s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, subject, serverExtData, effectiveRoles, claims)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Allow)
	require.Equal(t, []string{"no suitable rolebindings"}, result.Errors)
	require.Nil(t, result.BestEffectiveRole)
	require.Nil(t, result.VaultRules)
}

func Test_ReturnAllNeededPathsReturnNotNeedMFAOrApprovals(t *testing.T) {
	claims := Claims("100s", "200s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, subject, serverExtData, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allow)
	require.Equal(t, result.Errors, []string{})
	require.Equal(t, uuid1, result.BestEffectiveRole.RoleBindingUUID)
	require.Equal(t, sshVaultRules, result.VaultRules)
	require.Equal(t, "100s", result.TTL)
	require.Equal(t, "200s", result.MaxTTL)
}

func Test_ChooseBestWithNotNeedApprovals(t *testing.T) {
	claims := Claims("150s", "300s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, subject, serverExtData, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allow)
	require.Equal(t, uuid2, result.BestEffectiveRole.RoleBindingUUID)
	require.Equal(t, sshVaultRules, result.VaultRules)
}

func Test_ChooseBestWithBestApprovals(t *testing.T) {
	claims := Claims("250s", "500s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, subject, serverExtData, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allow)
	require.Equal(t, uuid3, result.BestEffectiveRole.RoleBindingUUID)
	require.Equal(t, sshVaultRules, result.VaultRules)
}

func Test_InvalidServer(t *testing.T) {
	claims := Claims("150s", "300s")
	claims["servers"] = []string{
		"0aaff1c0-0a93-4c15-9244-181aaeedd12d",
		"invalid_server_uuid",
	}
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, subject, serverExtData, effectiveRoles, claims)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Allow)
	require.Equal(t, []string{"servers are invalid: invalid_server_uuid"}, result.Errors)
	require.Nil(t, result.BestEffectiveRole)
	require.Nil(t, result.VaultRules)
}
