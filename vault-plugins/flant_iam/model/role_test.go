package model

import (
	"reflect"
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

func dieOnErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		panic(err)
	}
}

func checkDeepEqual(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("\nExpected:\n%#v\nGot:\n%#v\n", expected, got)
	}
}

var (
	role1 = Role{
		Name:          roleName1,
		Type:          GroupScopeProject,
		IncludedRoles: nil,
	}
	role2 = Role{
		Name:          roleName2,
		Type:          GroupScopeProject,
		IncludedRoles: nil,
	}
	role3 = Role{
		Name:          roleName3,
		Type:          GroupScopeProject,
		IncludedRoles: []IncludedRole{{Name: roleName1}},
	}
	role4 = Role{
		Name:          roleName4,
		Type:          GroupScopeProject,
		IncludedRoles: []IncludedRole{{Name: roleName1}, {Name: roleName2}},
	}
	role5 = Role{
		Name:          roleName5,
		Type:          GroupScopeProject,
		IncludedRoles: []IncludedRole{{Name: roleName2}, {Name: roleName3}},
	}
)

func Test_RoleDbSchema(t *testing.T) {
	schema := RoleSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("role schema is invalid: %v", err)
	}
}

func createRoles(t *testing.T, repo *RoleRepository, roles ...Role) {
	for _, role := range roles {
		tmp := role
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func prepareRepoForRolesTests(t *testing.T) *RoleRepository {
	schema, err := mergeSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil)
	dieOnErr(t, err)
	tx := store.Txn(true)
	repo := NewRoleRepository(tx)
	createRoles(t, repo, []Role{role1, role2, role3, role4, role5}...)
	return repo
}

func Test_Role_findDirectIncludingRoles(t *testing.T) {
	repoRole := prepareRepoForRolesTests(t)

	roles, err := repoRole.findDirectIncludingRoles(roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{roleName3: {}, roleName4: {}}, roles)
}

func Test_Role_FindAllIncludingRoles(t *testing.T) {
	repoRole := prepareRepoForRolesTests(t)

	roles, err := repoRole.FindAllIncludingRoles(roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{roleName3: {}, roleName4: {}, roleName5: {}}, roles)
}
