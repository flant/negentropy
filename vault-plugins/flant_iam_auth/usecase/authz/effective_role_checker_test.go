package authz

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
)

func Test_mapToEffectiveRoleResult(t *testing.T) {
	tx := usecase.RunFixtures(t, usecase.RoleFixture, usecase.TenantFixture, usecase.ProjectFixture).Txn(false)
	effectiveRoles := map[string][]usecase.EffectiveRole{
		fixtures.RoleName1: {usecase.EffectiveRole{RoleName: fixtures.RoleName1, RoleBindingUUID: "cbe03126-d3fb-49f1-b098-14a2840e5e0a", TenantUUID: fixtures.TenantUUID1, ValidTill: 0, RequireMFA: false, AnyProject: false, Projects: []string{"57ce1d2f-3991-4563-8713-b3129c0f6d93"}, NeedApprovals: 0, Options: map[string]interface{}{"k1": "v1"}}},
		fixtures.RoleName2: {
			usecase.EffectiveRole{RoleName: fixtures.RoleName2, RoleBindingUUID: "5a447c6c-9822-4875-a997-abdecff57121", TenantUUID: fixtures.TenantUUID2, ValidTill: 0, RequireMFA: false, AnyProject: false, Projects: []string{"57ce1d2f-3991-4563-8713-b3129c0f6d93"}, NeedApprovals: 0, Options: map[string]interface{}{"k2": "v2", "k3": "v3_1"}},
			usecase.EffectiveRole{RoleName: fixtures.RoleName2, RoleBindingUUID: "cbe03126-d3fb-49f1-b098-14a2840e5e0f", TenantUUID: fixtures.TenantUUID2, ValidTill: 0, RequireMFA: false, AnyProject: true, Projects: []string{}, NeedApprovals: 0, Options: map[string]interface{}{"k3": "v3_2", "k4": "v4"}},
			usecase.EffectiveRole{RoleName: fixtures.RoleName2, RoleBindingUUID: "cbe03126-d3fb-49f1-b098-14a2840e5e0g", TenantUUID: fixtures.TenantUUID2, ValidTill: 0, RequireMFA: false, AnyProject: true, Projects: []string{}, NeedApprovals: 0, Options: map[string]interface{}{"k3": "v3_3"}},
		},
		fixtures.RoleName8: {
			usecase.EffectiveRole{RoleName: fixtures.RoleName8, RoleBindingUUID: "cbe03126-d3fb-49f1-b098-14a2840e5e0a", TenantUUID: fixtures.TenantUUID1, ValidTill: 0, RequireMFA: false, AnyProject: false, Projects: []string{}, NeedApprovals: 0, Options: map[string]interface{}{"k5": "v5_1"}},
			usecase.EffectiveRole{RoleName: fixtures.RoleName8, RoleBindingUUID: "1be03126-d3fb-49f1-b098-14a2840e5e0a", TenantUUID: fixtures.TenantUUID1, ValidTill: 0, RequireMFA: false, AnyProject: false, Projects: []string{}, NeedApprovals: 0, Options: map[string]interface{}{"k5": "v5_2"}},
		},
	}
	checker := NewEffectiveRoleChecker(tx)

	results, err := checker.mapToEffectiveRoleResult(effectiveRoles)

	require.NoError(t, err)
	require.Equal(t, map[string]map[string]tenantResult{
		"RoleName1": {"00000001-0000-4000-A000-000000000000": {projects: map[string]EffectiveRoleProjectResult{"57ce1d2f-3991-4563-8713-b3129c0f6d93": {ProjectUUID: "57ce1d2f-3991-4563-8713-b3129c0f6d93", ProjectIdentifier: "", ProjectOptions: map[string][]interface{}{"k1": {"v1"}}, RequireMFA: false, NeedApprovals: false}}, tenantOptions: map[string][]interface{}{}}},
		"RoleName2": {"00000002-0000-4000-A000-000000000000": {projects: map[string]EffectiveRoleProjectResult{"00000000-0500-4000-A000-000000000000": {ProjectUUID: "00000000-0500-4000-A000-000000000000", ProjectIdentifier: "", ProjectOptions: map[string][]interface{}{}, RequireMFA: false, NeedApprovals: false}, "57ce1d2f-3991-4563-8713-b3129c0f6d93": {ProjectUUID: "57ce1d2f-3991-4563-8713-b3129c0f6d93", ProjectIdentifier: "", ProjectOptions: map[string][]interface{}{"k2": {"v2"}, "k3": {"v3_1"}}, RequireMFA: false, NeedApprovals: false}}, tenantOptions: map[string][]interface{}{"k3": {"v3_2", "v3_3"}, "k4": {"v4"}}}},
		"RoleName8": {"00000001-0000-4000-A000-000000000000": {projects: map[string]EffectiveRoleProjectResult{}, tenantOptions: map[string][]interface{}{"k5": {"v5_1", "v5_2"}}}},
	},
		results)
}
