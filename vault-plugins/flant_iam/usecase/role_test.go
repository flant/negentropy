package usecase

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/stretchr/testify/assert"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	roleName1  = "roleName1"
	roleName2  = "roleName2"
	roleName3  = "roleName3"
	roleName4  = "roleName4"
	roleName5  = "roleName5"
	roleName6  = "roleName6"
	roleName7  = "roleName7"
	roleName8  = "roleName8"
	roleName9  = "roleName9"
	roleName10 = "roleName10"
)

var (
	role1 = model.Role{
		Name:          roleName1,
		Scope:         model.RoleScopeProject,
		IncludedRoles: nil,
	}
	role2 = model.Role{
		Name:          roleName2,
		Scope:         model.RoleScopeProject,
		IncludedRoles: nil,
	}
	role3 = model.Role{
		Name:          roleName3,
		Scope:         model.RoleScopeProject,
		IncludedRoles: []model.IncludedRole{{Name: roleName1}},
	}
	role4 = model.Role{
		Name:          roleName4,
		Scope:         model.RoleScopeProject,
		IncludedRoles: []model.IncludedRole{{Name: roleName1}, {Name: roleName2}},
	}
	role5 = model.Role{
		Name:          roleName5,
		Scope:         model.RoleScopeProject,
		IncludedRoles: []model.IncludedRole{{Name: roleName2}, {Name: roleName3}},
	}
	role6 = model.Role{
		Name:  roleName6,
		Scope: model.RoleScopeProject,
	}
	role7 = model.Role{
		Name:  roleName7,
		Scope: model.RoleScopeProject,
	}
	role8 = model.Role{
		Name:  roleName8,
		Scope: model.RoleScopeTenant,
	}
	role9 = model.Role{
		Name:  roleName9,
		Scope: model.RoleScopeTenant,
	}
	role10 = model.Role{
		Name:          roleName10,
		Scope:         model.RoleScopeTenant,
		IncludedRoles: []model.IncludedRole{{Name: roleName9}},
	}
)

func createRoles(t *testing.T, repo *model.RoleRepository, roles ...model.Role) {
	for _, role := range roles {
		tmp := role
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func roleFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := model.NewRoleRepository(tx)
	createRoles(t, repo, []model.Role{role1, role2, role3, role4, role5, role6, role7, role8, role9, role10}...)
	err := tx.Commit()
	dieOnErr(t, err)
}



func Test_Role_findDirectIncludingRoles(t *testing.T) {
	tx := runFixtures(t, roleFixture).Txn(true)
	repo := model.NewRoleRepository(tx)

	roles, err := repo.FindDirectIncludingRoles(roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{roleName3: {}, roleName4: {}}, roles)
}

func Test_Role_FindAllIncludingRoles(t *testing.T) {
	tx := runFixtures(t, roleFixture).Txn(true)
	repo := model.NewRoleRepository(tx)

	roles, err := repo.FindAllIncludingRoles(roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{roleName3: {}, roleName4: {}, roleName5: {}}, roles)
}

func Test_includeRole(t *testing.T) {
	t.Run("adds sub-role to empty role", func(t *testing.T) {
		r := &model.Role{}
		sub := &model.IncludedRole{}

		includeRole(r, sub)

		assert.Contains(t, r.IncludedRoles, *sub)
	})

	t.Run("does not duplicate sub-roles", func(t *testing.T) {
		r := &model.Role{}
		sub := &model.IncludedRole{}

		includeRole(r, sub)
		includeRole(r, sub)

		assert.Contains(t, r.IncludedRoles, *sub)
		assert.Len(t, r.IncludedRoles, 1)
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

		assert.Contains(t, r.IncludedRoles, *sub1)
		assert.Contains(t, r.IncludedRoles, *sub2)
		assert.Contains(t, r.IncludedRoles, *sub3)
		assert.Len(t, r.IncludedRoles, 3)
		assert.Equal(t, "one", r.IncludedRoles[0].Name)
		assert.Equal(t, "two", r.IncludedRoles[1].Name)
		assert.Equal(t, "three", r.IncludedRoles[2].Name)
	})

	t.Run("updates options for the met name same", func(t *testing.T) {
		r := &model.Role{}
		sub1 := &model.IncludedRole{Name: "one", OptionsTemplate: "prev"}
		sub2 := &model.IncludedRole{Name: "one", OptionsTemplate: "new"}

		includeRole(r, sub1)
		includeRole(r, sub2)

		assert.NotContains(t, r.IncludedRoles, *sub1)
		assert.Contains(t, r.IncludedRoles, *sub2)
		assert.Len(t, r.IncludedRoles, 1)
		assert.Equal(t, "one", r.IncludedRoles[0].Name)
		assert.Equal(t, "new", r.IncludedRoles[0].OptionsTemplate)
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

		assert.Equal(t, r.IncludedRoles, expectedSubRoles)
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

		assert.Equal(t, r.IncludedRoles, expectedSubRoles)
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

		assert.Equal(t, r.IncludedRoles, expectedSubRoles)
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

		assert.Equal(t, r.IncludedRoles, expectedSubRoles)
	})
}
