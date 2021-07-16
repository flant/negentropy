package usecase

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	rbUUID1 = "00000000-0000-0001-0000-000000000000"
	rbUUID2 = "00000000-0000-0002-0000-000000000000"
	rbUUID3 = "00000000-0000-0003-0000-000000000000"
	rbUUID4 = "00000000-0000-0004-0000-000000000000"
	rbUUID5 = "00000000-0000-0005-0000-000000000000"
	// tenant_scoped_roles
	rbUUID6 = "00000000-0000-0006-0000-000000000000"
	rbUUID7 = "00000000-0000-0007-0000-000000000000"
	rbUUID8 = "00000000-0000-0008-0000-000000000000"
)

var (
	rb1 = model.RoleBinding{
		UUID:            rbUUID1,
		TenantUUID:      tenantUUID1,
		Identifier:      "rb1",
		ValidTill:       100,
		RequireMFA:      false,
		Users:           []string{userUUID1, userUUID2},
		Groups:          []string{groupUUID2, groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID1},
		AnyProject:      false,
		Projects:        []model.ProjectUUID{projectUUID1, projectUUID3},
		Roles: []model.BoundRole{{
			Name:    roleName1,
			Options: map[string]interface{}{"o1": "data1"},
		}},
		Origin: model.OriginIAM,
	}
	rb2 = model.RoleBinding{
		UUID:       rbUUID2,
		TenantUUID: tenantUUID2,
		Identifier: "rb2",
		ValidTill:  110,
		RequireMFA: false,
		Users:      []string{userUUID1, userUUID2},
		AnyProject: true,
		Projects:   nil,
		Roles: []model.BoundRole{{
			Name:    roleName1,
			Options: map[string]interface{}{"o1": "data2"},
		}},
		Origin: model.OriginIAM,
	}
	rb3 = model.RoleBinding{
		UUID:            rbUUID3,
		TenantUUID:      tenantUUID1,
		Identifier:      "rb3",
		ValidTill:       120,
		RequireMFA:      false,
		Users:           []string{userUUID2},
		Groups:          []string{groupUUID2, groupUUID5},
		ServiceAccounts: []string{serviceAccountUUID2},
		AnyProject:      true,
		Projects:        nil,
		Roles: []model.BoundRole{{
			Name:    roleName5,
			Options: map[string]interface{}{"o1": "data3"},
		}, {
			Name:    roleName7,
			Options: map[string]interface{}{"o1": "data4"},
		}},
		Origin: model.OriginIAM,
	}
	rb4 = model.RoleBinding{
		UUID:       rbUUID4,
		TenantUUID: tenantUUID1,
		Identifier: "rb4",
		ValidTill:  150,
		RequireMFA: false,
		Users:      []string{userUUID1},
		AnyProject: false,
		Projects:   []model.ProjectUUID{projectUUID3, projectUUID4},
		Roles: []model.BoundRole{{
			Name:    roleName8,
			Options: map[string]interface{}{"o1": "data5"},
		}},
		Origin: model.OriginIAM,
	}
	rb5 = model.RoleBinding{
		UUID:            rbUUID5,
		TenantUUID:      tenantUUID1,
		Identifier:      "rb5",
		ValidTill:       160,
		RequireMFA:      false,
		ServiceAccounts: []string{serviceAccountUUID1},
		AnyProject:      false,
		Projects:        []model.ProjectUUID{projectUUID3, projectUUID1},
		Roles: []model.BoundRole{{
			Name:    roleName1,
			Options: map[string]interface{}{"o1": "data6"},
		}},
		Origin: model.OriginIAM,
	}
	rb6 = model.RoleBinding{
		UUID:            rbUUID6,
		TenantUUID:      tenantUUID1,
		Identifier:      "rb6",
		ValidTill:       170,
		RequireMFA:      false,
		ServiceAccounts: []string{serviceAccountUUID2},
		AnyProject:      false,
		Projects:        nil,
		Roles: []model.BoundRole{{
			Name:    roleName9,
			Options: map[string]interface{}{"o1": "data7"},
		}},
		Origin: model.OriginIAM,
	}
	rb7 = model.RoleBinding{
		UUID:       rbUUID7,
		TenantUUID: tenantUUID1,
		Identifier: "rb7",
		ValidTill:  180,
		RequireMFA: false,
		Groups:     []model.GroupUUID{groupUUID4},
		AnyProject: false,
		Projects:   nil,
		Roles: []model.BoundRole{{
			Name:    roleName10,
			Options: map[string]interface{}{"o1": "data8"},
		}},
		Origin: model.OriginIAM,
	}
	rb8 = model.RoleBinding{
		UUID:       rbUUID8,
		TenantUUID: tenantUUID1,
		Identifier: "rb8",
		ValidTill:  190,
		RequireMFA: false,
		Users:      []model.UserUUID{userUUID2},
		AnyProject: false,
		Projects:   nil,
		Roles: []model.BoundRole{{
			Name:    roleName9,
			Options: map[string]interface{}{"o1": "data9"},
		}},
		Origin: model.OriginIAM,
	}
)

func createRoleBindings(t *testing.T, repo *model.RoleBindingRepository, rbs ...model.RoleBinding) {
	for _, rb := range rbs {
		tmp := rb
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func roleBindingFixture(t *testing.T, store *io.MemoryStore) {
	rbs := []model.RoleBinding{rb1, rb2, rb3, rb4, rb5, rb6, rb7, rb8}
	for i := range rbs {
		rbs[i].Members = appendMembers(makeMemberNotations(model.UserType, rbs[i].Users),
			makeMemberNotations(model.ServiceAccountType, rbs[i].ServiceAccounts),
			makeMemberNotations(model.GroupType, rbs[i].Groups))
	}
	tx := store.Txn(true)
	repo := model.NewRoleBindingRepository(tx)
	createRoleBindings(t, repo, rbs...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func roleBindingsUUIDSFromSlice(rbs []*model.RoleBinding) map[string]struct{} {
	uuids := map[string]struct{}{}
	for _, rb := range rbs {
		uuids[rb.UUID] = struct{}{}
	}
	return uuids
}

func roleBindingsUUIDsFromMap(rbs map[model.RoleBindingUUID]*model.RoleBinding) map[string]struct{} {
	uuids := map[string]struct{}{}
	for rbUUID := range rbs {
		uuids[rbUUID] = struct{}{}
	}
	return uuids
}

func Test_RoleBindingList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbs, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{
		rbUUID1: {}, rbUUID3: {}, rbUUID4: {}, rbUUID5: {},
		rbUUID6: {}, rbUUID7: {}, rbUUID8: {},
	},
		roleBindingsUUIDSFromSlice(rbs))
}

func Test_FindDirectRoleBindingsForTenantUser(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantUser(tenantUUID1, userUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID4: {}}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantServiceAccount(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantServiceAccount(tenantUUID1, serviceAccountUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID5: {}}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantGroups(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantGroups(tenantUUID1, groupUUID2, groupUUID3)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID3: {}}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForRoles(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForRoles(tenantUUID1, roleName1, roleName5, roleName8)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID3: {}, rbUUID4: {}, rbUUID5: {}}, roleBindingsUUIDsFromMap(rbsMap))
}

func Test_FindDirectRoleBindingsForTenantProject(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	repo := model.NewRoleBindingRepository(tx)

	rbsMap, err := repo.FindDirectRoleBindingsForTenantProject(tenantUUID1, projectUUID3)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID4: {}, rbUUID5: {}}, roleBindingsUUIDsFromMap(rbsMap))
}
