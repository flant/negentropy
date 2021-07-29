package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createRoleBindings(t *testing.T, repo *model.RoleBindingRepository, rbs ...model.RoleBinding) {
	for _, rb := range rbs {
		tmp := rb
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func roleBindingFixture(t *testing.T, store *io.MemoryStore) {
	rbs := fixtures.RoleBindings()
	for i := range rbs {
		rbs[i].Members = appendMembers(makeMemberNotations(model.UserType, rbs[i].Users),
			makeMemberNotations(model.ServiceAccountType, rbs[i].ServiceAccounts),
			makeMemberNotations(model.GroupType, rbs[i].Groups))
	}
	tx := store.Txn(true)
	repo := model.NewRoleBindingRepository(tx)
	createRoleBindings(t, repo, rbs...)
	err := tx.Commit()
	require.NoError(t, err)
}

func roleBindingsUUIDSFromSlice(rbs []*model.RoleBinding) []string {
	uuids := []string{}
	for _, rb := range rbs {
		uuids = append(uuids, rb.UUID)
	}
	return uuids
}

func roleBindingsUUIDsFromMap(rbs map[model.RoleBindingUUID]*model.RoleBinding) []string {
	uuids := []string{}
	for rbUUID := range rbs {
		uuids = append(uuids, rbUUID)
	}
	return uuids
}

func Test_RoleBindingList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbs, err := repo.List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		fixtures.RbUUID1, fixtures.RbUUID3, fixtures.RbUUID4, fixtures.RbUUID5,
		fixtures.RbUUID6, fixtures.RbUUID7, fixtures.RbUUID8,
	}, roleBindingsUUIDSFromSlice(rbs))
}

func Test_FindDirectRoleBindingsForTenantUser(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantUser(fixtures.TenantUUID1, fixtures.UserUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID4}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantServiceAccount(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantServiceAccount(fixtures.TenantUUID1, fixtures.ServiceAccountUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID5}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantGroups(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantGroups(fixtures.TenantUUID1, fixtures.GroupUUID2, fixtures.GroupUUID3)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID3}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForRoles(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForRoles(fixtures.TenantUUID1, fixtures.RoleName1, fixtures.RoleName5, fixtures.RoleName8)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID3, fixtures.RbUUID4, fixtures.RbUUID5},
		roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantProject(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantProject(fixtures.TenantUUID1, fixtures.ProjectUUID3)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RbUUID1, fixtures.RbUUID4, fixtures.RbUUID5},
		roleBindingsUUIDsFromMap(rbsMap))
}
