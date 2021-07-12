package model

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	roleName1 = "roleName1"
	roleName2 = "roleName2"
	roleName3 = "roleName3"
	roleName4 = "roleName4"
	roleName5 = "roleName5"
)

var (
	role1 = Role{
		Name:          roleName1,
		Scope:         RoleScopeProject,
		IncludedRoles: nil,
	}
	role2 = Role{
		Name:          roleName2,
		Scope:         RoleScopeProject,
		IncludedRoles: nil,
	}
	role3 = Role{
		Name:          roleName3,
		Scope:         RoleScopeProject,
		IncludedRoles: []IncludedRole{{Name: roleName1}},
	}
	role4 = Role{
		Name:          roleName4,
		Scope:         RoleScopeProject,
		IncludedRoles: []IncludedRole{{Name: roleName1}, {Name: roleName2}},
	}
	role5 = Role{
		Name:          roleName5,
		Scope:         RoleScopeProject,
		IncludedRoles: []IncludedRole{{Name: roleName2}, {Name: roleName3}},
	}
)

func createRoles(t *testing.T, repo *RoleRepository, roles ...Role) {
	for _, role := range roles {
		tmp := role
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func roleFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := NewRoleRepository(tx)
	createRoles(t, repo, []Role{role1, role2, role3, role4, role5}...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_RoleDbSchema(t *testing.T) {
	schema := RoleSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("role schema is invalid: %v", err)
	}
}

func Test_Role_findDirectIncludingRoles(t *testing.T) {
	tx := runFixtures(t, roleFixture).Txn(true)
	repo := NewRoleRepository(tx)

	roles, err := repo.findDirectIncludingRoles(roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{roleName3: {}, roleName4: {}}, roles)
}

func Test_Role_FindAllIncludingRoles(t *testing.T) {
	tx := runFixtures(t, roleFixture).Txn(true)
	repo := NewRoleRepository(tx)

	roles, err := repo.FindAllIncludingRoles(roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{roleName3: {}, roleName4: {}, roleName5: {}}, roles)
}
