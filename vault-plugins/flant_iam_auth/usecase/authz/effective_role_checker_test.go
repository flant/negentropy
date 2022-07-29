package authz

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
)

func Test_mapToEffectiveRoleResult(t *testing.T) {
	tx := usecase.RunFixtures(t, usecase.TenantFixture, usecase.ProjectFixture).Txn(false)
	effectiveRoles := map[string][]usecase.EffectiveRole{
		"ssh.open":         {usecase.EffectiveRole{RoleName: "ssh.open", RoleBindingUUID: "cbe03126-d3fb-49f1-b098-14a2840e5e0a", TenantUUID: fixtures.TenantUUID1, ValidTill: 0, RequireMFA: false, AnyProject: false, Projects: []string{"57ce1d2f-3991-4563-8713-b3129c0f6d93"}, NeedApprovals: 0, Options: map[string]interface{}(nil)}},
		"tenant.read":      {usecase.EffectiveRole{RoleName: "tenant.read", RoleBindingUUID: "5a447c6c-9822-4875-a997-abdecff57121", TenantUUID: fixtures.TenantUUID2, ValidTill: 0, RequireMFA: false, AnyProject: true, Projects: []string(nil), NeedApprovals: 0, Options: map[string]interface{}(nil)}},
		"tenant.read.auth": {usecase.EffectiveRole{RoleName: "tenant.read.auth", RoleBindingUUID: "cbe03126-d3fb-49f1-b098-14a2840e5e0a", TenantUUID: fixtures.TenantUUID1, ValidTill: 0, RequireMFA: false, AnyProject: true, Projects: []string{"57ce1d2f-3991-4563-8713-b3129c0f6d93"}, NeedApprovals: 0, Options: map[string]interface{}(nil)}}}
	checker := NewEffectiveRoleChecker(tx)

	results, err := checker.mapToEffectiveRoleResult(effectiveRoles)

	require.NoError(t, err)
	require.Equal(t, map[string]map[string]map[string]EffectiveRoleProjectResult{
		"ssh.open":         {"00000001-0000-0000-0000-000000000000": {"57ce1d2f-3991-4563-8713-b3129c0f6d93": {ProjectUUID: "57ce1d2f-3991-4563-8713-b3129c0f6d93", ProjectIdentifier: "", RequireMFA: false, NeedApprovals: false}}},
		"tenant.read":      {"00000002-0000-0000-0000-000000000000": {"00000000-0500-0000-0000-000000000000": {ProjectUUID: "00000000-0500-0000-0000-000000000000", ProjectIdentifier: "", RequireMFA: false, NeedApprovals: false}}},
		"tenant.read.auth": {"00000001-0000-0000-0000-000000000000": {"00000000-0100-0000-0000-000000000000": {ProjectUUID: "00000000-0100-0000-0000-000000000000", ProjectIdentifier: "", RequireMFA: false, NeedApprovals: false}, "00000000-0200-0000-0000-000000000000": {ProjectUUID: "00000000-0200-0000-0000-000000000000", ProjectIdentifier: "", RequireMFA: false, NeedApprovals: false}, "00000000-0300-0000-0000-000000000000": {ProjectUUID: "00000000-0300-0000-0000-000000000000", ProjectIdentifier: "", RequireMFA: false, NeedApprovals: false}, "00000000-0400-0000-0000-000000000000": {ProjectUUID: "00000000-0400-0000-0000-000000000000", ProjectIdentifier: "", RequireMFA: false, NeedApprovals: false}}}}, results)
}
