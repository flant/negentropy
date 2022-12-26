package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
)

func Test_TenantList(t *testing.T) {
	tx := RunFixtures(t, TenantFixture).Txn(true)
	repo := iam_repo.NewTenantRepository(tx)

	tenants, err := repo.List(false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range tenants {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.TenantUUID1, fixtures.TenantUUID2}, ids)
}

func Test_TenantCascadeEraseAfterCascadeDelete(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture,
		ProjectFixture, RoleFixture, RoleBindingFixture,
	).Txn(true)
	tenantService := Tenants(tx, "")
	err := tenantService.Delete(fixtures.TenantUUID1)
	require.NoError(t, err)
	// ====
	err = tenantService.CascadeErase(fixtures.TenantUUID1)
	require.NoError(t, err)
	// ====
	// Checks all are erased
	tenants, err := iam_repo.NewTenantRepository(tx).List(true)
	require.NoError(t, err)
	require.Equal(t, 1, len(tenants)) // only second tenant

	users, err := iam_repo.NewUserRepository(tx).List(fixtures.TenantUUID1, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(users))

	sas, err := iam_repo.NewServiceAccountRepository(tx).List(fixtures.TenantUUID1, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(sas))

	groups, err := iam_repo.NewGroupRepository(tx).List(fixtures.TenantUUID1, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(groups))

	projects, err := iam_repo.NewProjectRepository(tx).List(fixtures.TenantUUID1, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(projects))

	rbs, err := iam_repo.NewRoleBindingRepository(tx).List(fixtures.TenantUUID1, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(rbs))
}
