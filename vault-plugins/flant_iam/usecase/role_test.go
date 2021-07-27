package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createRoles(t *testing.T, repo *model.RoleRepository, roles ...model.Role) {
	for _, role := range roles {
		tmp := role
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func roleFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := model.NewRoleRepository(tx)
	createRoles(t, repo, fixtures.Roles()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_Role_findDirectIncludingRoles(t *testing.T) {
	tx := runFixtures(t, roleFixture).Txn(true)
	repo := model.NewRoleRepository(tx)

	roles, err := repo.FindDirectIncludingRoles(fixtures.RoleName1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RoleName3, fixtures.RoleName4}, stringSlice(roles))
}

func Test_Role_FindAllIncludingRoles(t *testing.T) {
	tx := runFixtures(t, roleFixture).Txn(true)
	repo := model.NewRoleRepository(tx)

	roles, err := repo.FindAllIncludingRoles(fixtures.RoleName1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.RoleName3, fixtures.RoleName4, fixtures.RoleName5}, stringSlice(roles))
}

func Test_includeRole(t *testing.T) {
	t.Run("adds sub-role to empty role", func(t *testing.T) {
		r := &model.Role{}
		sub := &model.IncludedRole{}

		includeRole(r, sub)

		require.Contains(t, r.IncludedRoles, *sub)
	})

	t.Run("does not duplicate sub-roles", func(t *testing.T) {
		r := &model.Role{}
		sub := &model.IncludedRole{}

		includeRole(r, sub)
		includeRole(r, sub)

		require.Contains(t, r.IncludedRoles, *sub)
		require.Len(t, r.IncludedRoles, 1)
	})

	t.Run("does not duplicate sub-roles based by name", func(t *testing.T) {
		r := &model.Role{}
		sub1 := &model.IncludedRole{Name: "one"}
		sub2 := &model.IncludedRole{Name: "two"}
		sub3 := &model.IncludedRole{Name: "three"}
		sub11 := &model.IncludedRole{Name: "two"}

		includeRole(r, sub1)
		includeRole(r, sub2)
		includeRole(r, sub3)
		includeRole(r, sub11)

		require.Contains(t, r.IncludedRoles, *sub1)
		require.Contains(t, r.IncludedRoles, *sub2)
		require.Contains(t, r.IncludedRoles, *sub3)
		require.Len(t, r.IncludedRoles, 3)
		require.Equal(t, "one", r.IncludedRoles[0].Name)
		require.Equal(t, "two", r.IncludedRoles[1].Name)
		require.Equal(t, "three", r.IncludedRoles[2].Name)
	})

	t.Run("updates options for the met name same", func(t *testing.T) {
		r := &model.Role{}
		sub1 := &model.IncludedRole{Name: "one", OptionsTemplate: "prev"}
		sub2 := &model.IncludedRole{Name: "one", OptionsTemplate: "new"}

		includeRole(r, sub1)
		includeRole(r, sub2)

		require.NotContains(t, r.IncludedRoles, *sub1)
		require.Contains(t, r.IncludedRoles, *sub2)
		require.Len(t, r.IncludedRoles, 1)
		require.Equal(t, "one", r.IncludedRoles[0].Name)
		require.Equal(t, "new", r.IncludedRoles[0].OptionsTemplate)
	})
}

func Test_excludeRole(t *testing.T) {
	t.Run("empty values do nothing", func(t *testing.T) {
		r := &model.Role{}

		excludeRole(r, "")
	})

	t.Run("name mismatch does nothing", func(t *testing.T) {
		subRoles := []model.IncludedRole{{Name: "one", OptionsTemplate: "<>"}}
		expectedSubRoles := make([]model.IncludedRole, len(subRoles))
		copy(expectedSubRoles, subRoles)
		r := &model.Role{IncludedRoles: subRoles}

		excludeRole(r, "zz")

		require.Equal(t, r.IncludedRoles, expectedSubRoles)
	})

	t.Run("name match removes sub-role from the start", func(t *testing.T) {
		subRoles := []model.IncludedRole{
			{Name: "one", OptionsTemplate: "<1>"},
			{Name: "two", OptionsTemplate: "<2>"},
			{Name: "three", OptionsTemplate: "<3>"},
		}
		expectedSubRoles := make([]model.IncludedRole, len(subRoles)-1)
		copy(expectedSubRoles, subRoles[1:])
		r := &model.Role{IncludedRoles: subRoles}

		excludeRole(r, "one")

		require.Equal(t, r.IncludedRoles, expectedSubRoles)
	})

	t.Run("name match removes sub-role from the end", func(t *testing.T) {
		subRoles := []model.IncludedRole{
			{Name: "one", OptionsTemplate: "<1>"},
			{Name: "two", OptionsTemplate: "<2>"},
			{Name: "three", OptionsTemplate: "<3>"},
		}
		expectedSubRoles := make([]model.IncludedRole, len(subRoles)-1)
		copy(expectedSubRoles, subRoles[:2])
		r := &model.Role{IncludedRoles: subRoles}

		excludeRole(r, "three")

		require.Equal(t, r.IncludedRoles, expectedSubRoles)
	})

	t.Run("name match removes sub-role from the middle", func(t *testing.T) {
		subRoles := []model.IncludedRole{
			{Name: "one", OptionsTemplate: "<1>"},
			{Name: "two", OptionsTemplate: "<2>"},
			{Name: "three", OptionsTemplate: "<3>"},
		}
		expectedSubRoles := make([]model.IncludedRole, 0)
		expectedSubRoles = append(expectedSubRoles, subRoles[0], subRoles[2])
		r := &model.Role{IncludedRoles: subRoles}

		excludeRole(r, "two")

		require.Equal(t, r.IncludedRoles, expectedSubRoles)
	})
}

func Test_Role_IsArchived(t *testing.T) {
	tx := runFixtures(t, roleFixture).Txn(true)
	err := (&RoleService{tx}).Delete(fixtures.RoleName1, time.Now().Unix(), 1)
	require.NoError(t, err)

	role, err := (&RoleService{tx}).Get(fixtures.RoleName1)
	require.NoError(t, err)
	require.NotNil(t, role)
	require.Greater(t, role.ArchivingTimestamp, int64(0))
}
