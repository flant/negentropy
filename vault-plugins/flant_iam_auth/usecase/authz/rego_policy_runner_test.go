package authz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
)

var (
	uuid1          = "uuid1"
	uuid2          = "uuid2"
	uuid3          = "uuid3"
	uuid4          = "uuid4"
	effectiveRoles = []iam_usecase.EffectiveRole{
		{ // not need MFA or approvals
			RoleName:        "ssh",
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
			RoleName:        "ssh",
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
			RoleName:        "ssh",
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
			RoleName:        "ssh",
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
	sshPolicy = `
package negentropy


default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

filtered_bindings[r] {
	tenant := input.tenant_uuid
    project := input.project_uuid
	some i
	r := data.effective_roles[i]
    	data.effective_roles[i].tenant_uuid==tenant
    	data.effective_roles[i].projects[_]==project
        to_seconds_number(data.effective_roles[i].options.ttl)>=to_seconds_number(requested_ttl)
        to_seconds_number(data.effective_roles[i].options.max_ttl)>=to_seconds_number(requested_max_ttl)
}

default allow = false

allow {count(filtered_bindings) >0}

# пути по которым должен появится доступ
rules = [
	{"path":"ssh/sign/signer","capabilities":["update"]},
    {"path":"auth/flant_iam_auth/multipass_owner","capabilities":["read"]},
    {"path":"auth/flant_iam_auth/query_server","capabilities":["read"]},
    {"path":"auth/flant_iam_auth/tenant/*","capabilities":["read","list"]}
    ]{allow}

ttl := requested_ttl {allow}

max_ttl := requested_max_ttl {allow}

# Переводим в число секунд
to_seconds_number(t) = x {
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

	sshVaultRules = []Rule{
		{
			Path:   "ssh/sign/signer",
			Update: true,
		}, {
			Path: "auth/flant_iam_auth/multipass_owner",
			Read: true,
		}, {
			Path: "auth/flant_iam_auth/query_server", // TODO
			Read: true,
		}, {
			Path: "auth/flant_iam_auth/tenant/*", // TODO  split for tenant_list and others
			Read: true,
			List: true,
		},
	}
)

func Claims(ttl, maxTTL string) map[string]interface{} {
	return map[string]interface{}{
		//"role":         "ssh",
		"tenant_uuid":  "t1",
		"project_uuid": "p1",
		"ttl":          ttl,
		"max_ttl":      maxTTL,
	}
}

func Test_NotAllowedDueToBigTTL(t *testing.T) {
	claims := Claims("10000s", "100s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, UserData{}, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Allow)
	require.Nil(t, result.BestEffectiveRole)
	require.Nil(t, result.VaultRules)
}

func Test_NotAllowedDueToBigMaxTTL(t *testing.T) {
	claims := Claims("100s", "10000s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, UserData{}, effectiveRoles, claims)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Allow)
	require.Nil(t, result.BestEffectiveRole)
	require.Nil(t, result.VaultRules)
}

func Test_ReturnAllNeededPathsReturnNotNeedMFAOrApprovals(t *testing.T) {
	claims := Claims("100s", "200s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, UserData{}, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allow)
	require.Equal(t, uuid1, result.BestEffectiveRole.RoleBindingUUID)
	require.Equal(t, sshVaultRules, result.VaultRules)
	require.Equal(t, "100s", result.TTL)
	require.Equal(t, "200s", result.MaxTTL)
}

func Test_ChooseBestWithNotNeedApprovals(t *testing.T) {
	claims := Claims("150s", "300s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, UserData{}, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allow)
	require.Equal(t, uuid2, result.BestEffectiveRole.RoleBindingUUID)
	require.Equal(t, sshVaultRules, result.VaultRules)
}

func Test_ChooseBestWithBestApprovals(t *testing.T) {
	claims := Claims("250s", "500s")
	ctx := context.TODO()

	result, err := ApplyRegoPolicy(ctx, sshPolicy, UserData{}, effectiveRoles, claims)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allow)
	require.Equal(t, uuid3, result.BestEffectiveRole.RoleBindingUUID)
	require.Equal(t, sshVaultRules, result.VaultRules)
}
