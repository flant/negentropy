package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	rbUUID1 = "00000000-0000-0001-0000-000000000000"
	rbUUID2 = "00000000-0000-0002-0000-000000000000"
	rbUUID3 = "00000000-0000-0003-0000-000000000000"
	rbUUID4 = "00000000-0000-0004-0000-000000000000"
)

var (
	rb1 = RoleBinding{
		UUID:            rbUUID1,
		TenantUUID:      tenantUUID1,
		Version:         "",
		ValidTill:       100,
		RequireMFA:      false,
		Users:           []string{userUUID1, userUUID2},
		Groups:          []string{groupUUID2, groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID1},
		Subjects:        nil,
		AnyProject:      false,
		Projects:        []ProjectUUID{projectUUID1, projectUUID3},
		Roles: []BoundRole{{
			Name:    roleName1,
			Scope:   RoleScopeProject,
			Options: map[string]interface{}{"o1": "data1"},
		}},
		Origin:     OriginIAM,
		Extensions: nil,
	}
	rb2 = RoleBinding{
		UUID:       rbUUID2,
		TenantUUID: tenantUUID2,
		Version:    "",
		ValidTill:  100,
		RequireMFA: false,
		Users:      []string{userUUID1, userUUID2},
		Subjects:   nil,
		AnyProject: true,
		Projects:   nil,
		Roles: []BoundRole{{
			Name:    roleName1,
			Scope:   RoleScopeProject,
			Options: map[string]interface{}{"o1": "data2"},
		}},
		Origin:     OriginIAM,
		Extensions: nil,
	}
	rb3 = RoleBinding{
		UUID:            rbUUID3,
		TenantUUID:      tenantUUID1,
		Version:         "",
		ValidTill:       200,
		RequireMFA:      false,
		Users:           []string{userUUID2},
		Groups:          []string{groupUUID2, groupUUID5},
		ServiceAccounts: []string{serviceAccountUUID2},
		Subjects:        nil,
		AnyProject:      true,
		Projects:        nil,
		Roles: []BoundRole{{
			Name:    roleName5,
			Scope:   RoleScopeProject,
			Options: map[string]interface{}{"o1": "data3"},
		}, {
			Name:    roleName7,
			Scope:   RoleScopeProject,
			Options: map[string]interface{}{"o1": "data4"},
		}},
		Origin:     OriginIAM,
		Extensions: nil,
	}
	rb4 = RoleBinding{
		UUID:       rbUUID4,
		TenantUUID: tenantUUID1,
		Version:    "",
		ValidTill:  150,
		RequireMFA: false,
		Users:      []string{userUUID1},
		Subjects:   nil,
		AnyProject: false,
		Projects:   []ProjectUUID{projectUUID3, projectUUID4},
		Roles: []BoundRole{{
			Name:    roleName8,
			Scope:   RoleScopeProject,
			Options: map[string]interface{}{"o1": "data5"},
		}},
		Origin:     OriginIAM,
		Extensions: nil,
	}
)

func createRoleBindings(t *testing.T, repo *RoleBindingRepository, rbs ...RoleBinding) {
	for _, rb := range rbs {
		tmp := rb
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func roleBindingFixture(t *testing.T, store *io.MemoryStore) {
	rbs := []RoleBinding{rb1, rb2, rb3, rb4}
	for i := range rbs {
		rbs[i].Subjects = appendSubjects(makeSubjectNotations(UserType, rbs[i].Users),
			makeSubjectNotations(ServiceAccountType, rbs[i].ServiceAccounts),
			makeSubjectNotations(GroupType, rbs[i].Groups))
	}
	tx := store.Txn(true)
	repo := NewRoleBindingRepository(tx)
	createRoleBindings(t, repo, rbs...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_RoleBindingDbSchema(t *testing.T) {
	schema := RoleBindingSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("role binding schema is invalid: %v", err)
	}
}

func Test_RoleBindingOnCreationSubjectsCalculation(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	ten := &tenant1
	sa1 := &sa1
	sa2 := &sa2
	u1 := &user1
	u2 := &user2
	g1 := &group1
	g2 := &group2

	tests := []struct {
		name       string
		subjects   []Model
		assertions func(*testing.T, *RoleBinding)
	}{
		{
			name:     "no subjects",
			subjects: []Model{},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 0, "should contain no serviceaccounts")
				assert.Len(t, rb.Users, 0, "should contain no users")
				assert.Len(t, rb.Groups, 0, "should contain no groups")
			},
		},
		{
			name:     "single SA",
			subjects: []Model{sa1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 1, "should contain 1 serviceaccount")
				assert.Len(t, rb.Users, 0, "should contain no users")
				assert.Len(t, rb.Groups, 0, "should contain no groups")
				assert.Contains(t, rb.ServiceAccounts, sa1.UUID)
			},
		},
		{
			name:     "single user",
			subjects: []Model{u1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 0, "should contain no serviceaccounts")
				assert.Len(t, rb.Users, 1, "should contain 1 user")
				assert.Len(t, rb.Groups, 0, "should contain no groups")
				assert.Contains(t, rb.Users, u1.UUID)
			},
		},
		{
			name:     "single group",
			subjects: []Model{g1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 0, "should contain no serviceaccounts")
				assert.Len(t, rb.Users, 0, "should contain no users")
				assert.Len(t, rb.Groups, 1, "should contain 1 group")
				assert.Contains(t, rb.Groups, g1.UUID)
			},
		},
		{
			name:     "all by one",
			subjects: []Model{g1, u1, sa1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 1, "should contain 1 serviceaccount")
				assert.Len(t, rb.Users, 1, "should contain 1 user")
				assert.Len(t, rb.Groups, 1, "should contain 1 group")
				assert.Contains(t, rb.ServiceAccounts, sa1.UUID)
				assert.Contains(t, rb.Users, u1.UUID)
				assert.Contains(t, rb.Groups, g1.UUID)
			},
		},
		{
			name:     "keeps addition ordering",
			subjects: []Model{sa1, g2, u1, sa2, u2, g1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 2, "should contain 2 serviceaccount")
				assert.Len(t, rb.Users, 2, "should contain 2 user")
				assert.Len(t, rb.Groups, 2, "should contain 2 group")

				assert.Equal(t, rb.ServiceAccounts, []ServiceAccountUUID{sa1.UUID, sa2.UUID})
				assert.Equal(t, rb.Groups, []GroupUUID{g2.UUID, g1.UUID})
				assert.Equal(t, rb.Users, []UserUUID{u1.UUID, u2.UUID})
			},
		},

		{
			name:     "ignores duplicates",
			subjects: []Model{sa1, g2, u1, sa2, u2, sa1, g1, u1, g2},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 2, "should contain 2 serviceaccount")
				assert.Len(t, rb.Users, 2, "should contain 2 user")
				assert.Len(t, rb.Groups, 2, "should contain 2 group")

				assert.Equal(t, rb.ServiceAccounts, []ServiceAccountUUID{sa1.UUID, sa2.UUID})
				assert.Equal(t, rb.Groups, []GroupUUID{g2.UUID, g1.UUID})
				assert.Equal(t, rb.Users, []UserUUID{u1.UUID, u2.UUID})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := &RoleBinding{
				UUID:       uuid.New(),
				TenantUUID: ten.UUID,
				Origin:     OriginIAM,
				Subjects:   toSubjectNotations(tt.subjects...),

				ServiceAccounts: []ServiceAccountUUID{"nonsense"},
				Users:           []UserUUID{"nonsense"},
				Groups:          []GroupUUID{"nonsense"},
			}
			if err := NewRoleBindingRepository(tx).Create(rb); err != nil {
				t.Fatalf("cannot create: %v", err)
			}

			created, err := NewRoleBindingRepository(tx).GetByID(rb.UUID)
			if err != nil {
				t.Fatalf("cannot get: %v", err)
			}

			tt.assertions(t, created)
		})
	}
}

func roleBindingsUUIDSFromSlice(rbs []*RoleBinding) map[string]struct{} {
	uuids := map[string]struct{}{}
	for _, rb := range rbs {
		uuids[rb.UUID] = struct{}{}
	}
	return uuids
}

func roleBindingsUUIDsFromMap(rbs map[RoleBindingUUID]*RoleBinding) map[string]struct{} {
	uuids := map[string]struct{}{}
	for rbUUID := range rbs {
		uuids[rbUUID] = struct{}{}
	}
	return uuids
}

func Test_RoleBindingList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := NewRoleBindingRepository(tx)

	rbs, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID3: {}, rbUUID4: {}}, roleBindingsUUIDSFromSlice(rbs))
}

func Test_FindDirectRoleBindingsForTenantUser(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantUser(tenantUUID1, userUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID4: {}}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantServiceAccount(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantServiceAccount(tenantUUID1, serviceAccountUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantGroups(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantGroups(tenantUUID1, groupUUID2, groupUUID3)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID3: {}}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForRoles(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := NewRoleBindingRepository(tx)

	rbsSet, err := repo.FindDirectRoleBindingsForRoles(tenantUUID1, roleName1, roleName5, roleName8)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID3: {}, rbUUID4: {}}, rbsSet)
}

func Test_FindDirectRoleBindingsForTenantProject(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := NewRoleBindingRepository(tx)

	rbsSet, err := repo.FindDirectRoleBindingsForTenantProject(tenantUUID1, projectUUID3)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID4: {}}, rbsSet)
}
