package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
)

func Test_collectAllRolesAndRoleBindings(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	roles, roleBindings, err := rr.collectAllRolesAndRoleBindings(fixtures.RoleName1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RoleName1, fixtures.RoleName3, fixtures.RoleName4, fixtures.RoleName5},
		roleNames(roles))
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID2, fixtures.RbUUID3, fixtures.RbUUID5},
		roleBindingsUUIDsFromMap(roleBindings))
}

func Test_collectAllRoleBindingsForUser(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	roleBindings, err := rr.collectAllRoleBindingsForUser(fixtures.UserUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID2, fixtures.RbUUID3, fixtures.RbUUID4, fixtures.RbUUID7},
		roleBindingsUUIDsFromMap(roleBindings))
}

func Test_CheckUserForProjectScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	hasRole, effectiveRoles, err := rr.CheckUserForProjectScopedRole(fixtures.UserUUID1, fixtures.RoleName1,
		fixtures.ProjectUUID1)

	require.NoError(t, err)
	require.True(t, hasRole)
	expectedEffectiveRoles := []EffectiveRole{}
	//RoleBindingParams{ValidTill: 120, RequireMFA: false, Options: map[string]interface{}{"o1": "data3"}}
	require.Equal(t, expectedEffectiveRoles, effectiveRoles)
}

func Test_collectAllRoleBindingsForServiceAccount(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	roleBindings, err := rr.collectAllRoleBindingsForServiceAccount(fixtures.ServiceAccountUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID3, fixtures.RbUUID5, fixtures.RbUUID7},
		roleBindingsUUIDsFromMap(roleBindings))
}

func Test_CheckServiceAccountForProjectScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	hasRole, effectiveRoles, err := rr.CheckServiceAccountForProjectScopedRole(fixtures.ServiceAccountUUID1, fixtures.RoleName1,
		fixtures.ProjectUUID1)

	require.NoError(t, err)
	require.True(t, hasRole)
	expectedEffectiveRoles := []EffectiveRole{}
	//expectedParams := RoleBindingParams{ValidTill: 160, RequireMFA: false, Options: map[string]interface{}{"o1": "data6"}}
	require.Equal(t, expectedEffectiveRoles, effectiveRoles)
}

func Test_CheckUserForTenantScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	hasRole, effectiveRoles, err := rr.CheckUserForTenantScopedRole(fixtures.UserUUID2, fixtures.RoleName9,
		fixtures.TenantUUID1)

	require.NoError(t, err)
	require.True(t, hasRole)
	expectedEffectiveRoles := []EffectiveRole{}
	//expectedParams := RoleBindingParams{ValidTill: 190, RequireMFA: false, Options: map[string]interface{}{"o1": "data9"}}
	require.Equal(t, expectedEffectiveRoles, effectiveRoles)
}

func Test_CheckServiceAccountForTenantScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	hasRole, effectiveRoles, err := rr.CheckServiceAccountForTenantScopedRole(fixtures.ServiceAccountUUID2,
		fixtures.RoleName9, fixtures.TenantUUID1)

	require.NoError(t, err)
	require.True(t, hasRole)
	expectedEffectiveRoles := []EffectiveRole{}
	//expectedParams := RoleBindingParams{ValidTill: 180, RequireMFA: false, Options: map[string]interface{}{"o1": "data8"}}
	require.Equal(t, expectedEffectiveRoles, effectiveRoles)
}

func Test_FindMembersWithProjectScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	users, serviceAccounts, err := rr.FindMembersWithProjectScopedRole(fixtures.RoleName1, fixtures.TenantUUID1, fixtures.ProjectUUID3)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.UserUUID1, fixtures.UserUUID2, fixtures.UserUUID3, fixtures.UserUUID4},
		users)
	require.ElementsMatch(t, []string{fixtures.ServiceAccountUUID1, fixtures.ServiceAccountUUID2}, serviceAccounts)
}

func Test_FindMembersWithTenantScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
	}

	users, serviceAccounts, err := rr.FindMembersWithTenantScopedRole(fixtures.RoleName9, fixtures.TenantUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.UserUUID1, fixtures.UserUUID2, fixtures.UserUUID3, fixtures.UserUUID4},
		users)
	require.ElementsMatch(t, []string{
		fixtures.ServiceAccountUUID1, fixtures.ServiceAccountUUID2, fixtures.ServiceAccountUUID3,
	}, serviceAccounts)
}
