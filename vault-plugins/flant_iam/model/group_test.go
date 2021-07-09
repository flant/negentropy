package model

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	origin              = "flant_iam"
	groupUUID1          = "00000000-0001-0000-0000-000000000000"
	groupUUID2          = "00000000-0002-0000-0000-000000000000"
	groupUUID3          = "00000000-0003-0000-0000-000000000000"
	groupUUID4          = "00000000-0004-0000-0000-000000000000"
	groupUUID5          = "00000000-0005-0000-0000-000000000000"
	userUUID1           = "00000000-0000-0000-0000-000000000001"
	userUUID2           = "00000000-0000-0000-0000-000000000002"
	userUUID3           = "00000000-0000-0000-0000-000000000003"
	userUUID4           = "00000000-0000-0000-0000-000000000004"
	serviceAccountUUID1 = "00000000-0003-0000-0000-000000000011"
	serviceAccountUUID2 = "00000000-0003-0000-0000-000000000012"
	serviceAccountUUID3 = "00000000-0003-0000-0000-000000000013"
)

var (
	group1 = Group{
		UUID:            groupUUID1,
		TenantUUID:      tenantUUID1,
		Users:           []string{userUUID2, userUUID3},
		Groups:          []string{groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID1},
		Origin:          origin,
	}
	group2 = Group{
		UUID:       groupUUID2,
		TenantUUID: tenantUUID1,
		Users:      []string{userUUID1, userUUID3},

		Origin: origin,
	}
	group3 = Group{
		UUID:            groupUUID3,
		TenantUUID:      tenantUUID2,
		Users:           []string{userUUID3, userUUID4},
		ServiceAccounts: []string{serviceAccountUUID1},
		Origin:          origin,
	}
	group4 = Group{
		UUID:            groupUUID4,
		TenantUUID:      tenantUUID1,
		Users:           []string{userUUID2, userUUID3},
		Groups:          []string{groupUUID2, groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID2, serviceAccountUUID3},
		Origin:          origin,
	}
	group5 = Group{
		UUID:       groupUUID5,
		TenantUUID: tenantUUID1,
		Groups:     []string{groupUUID2, groupUUID1},
		Origin:     origin,
	}
)

func createGroups(t *testing.T, repo *GroupRepository, groups ...Group) {
	for _, group := range groups {
		tmp := group
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func Test_GroupDbSchema(t *testing.T) {
	schema := GroupSchema()
	if err := schema.Validate(); err != nil {
		t.
			Fatalf("group schema is invalid: %v", err)
	}
}

func prepareRepo(t *testing.T) *GroupRepository {
	schema, err := mergeSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil)
	dieOnErr(t, err)
	tx := store.Txn(true)
	repoTenant := NewTenantRepository(tx)
	createTenants(t, repoTenant, []Tenant{tenant1, tenant2}...)
	repo := NewGroupRepository(tx)
	createGroups(t, repo, []Group{group1, group2, group3, group4, group5}...)
	return repo
}

func Test_ListGroups(t *testing.T) {
	repo := prepareRepo(t)
	ids, err := repo.List(tenantUUID1)
	dieOnErr(t, err)
	checkDeepEqual(t, []string{groupUUID1, groupUUID2, groupUUID4, groupUUID5}, ids)
}

func Test_findDirectParentGroupsByUserUUID(t *testing.T) {
	repo := prepareRepo(t)
	ids, err := repo.findDirectParentGroupsByUserUUID(tenantUUID1, userUUID3)
	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}, groupUUID2: {}, groupUUID4: {}}, ids)
}

func Test_findDirectParentGroupsByServiceAccountUUID(t *testing.T) {
	repo := prepareRepo(t)
	ids, err := repo.findDirectParentGroupsByServiceAccountUUID(tenantUUID1, serviceAccountUUID1)
	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}}, ids)
}

func Test_findDirectParentGroupsByGroupUUID(t *testing.T) {
	repo := prepareRepo(t)
	ids, err := repo.findDirectParentGroupsByGroupUUID(tenantUUID1, groupUUID1)
	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID5: {}}, ids)
}

func Test_FindAllParentGroupsForUserUUID(t *testing.T) {
	repo := prepareRepo(t)
	ids, err := repo.FindAllParentGroupsForUserUUID(tenantUUID1, userUUID1)
	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID2: {}, groupUUID4: {}, groupUUID5: {}}, ids)
}

func Test_FindAllParentGroupsForServiceAccountUUID(t *testing.T) {
	repo := prepareRepo(t)
	ids, err := repo.FindAllParentGroupsForServiceAccountUUID(tenantUUID1, serviceAccountUUID1)
	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}, groupUUID5: {}}, ids)
}
