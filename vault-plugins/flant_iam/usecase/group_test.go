package usecase

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	groupUUID1 = "00000000-0001-0000-0000-000000000000"
	groupUUID2 = "00000000-0002-0000-0000-000000000000"
	groupUUID3 = "00000000-0003-0000-0000-000000000000"
	groupUUID4 = "00000000-0004-0000-0000-000000000000"
	groupUUID5 = "00000000-0005-0000-0000-000000000000"
)

var (
	group1 = model.Group{
		UUID:            groupUUID1,
		TenantUUID:      tenantUUID1,
		Users:           []string{userUUID2, userUUID3},
		Groups:          []string{groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID1},
		Origin:          model.OriginIAM,
	}
	group2 = model.Group{
		UUID:       groupUUID2,
		TenantUUID: tenantUUID1,
		Users:      []string{userUUID1, userUUID3},
		Origin:     model.OriginIAM,
	}
	group3 = model.Group{
		UUID:            groupUUID3,
		TenantUUID:      tenantUUID2,
		Users:           []string{userUUID3, userUUID4},
		ServiceAccounts: []string{serviceAccountUUID1},
		Origin:          model.OriginIAM,
	}
	group4 = model.Group{
		UUID:            groupUUID4,
		TenantUUID:      tenantUUID1,
		Users:           []string{userUUID2, userUUID3},
		Groups:          []string{groupUUID2, groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID2, serviceAccountUUID3},
		Origin:          model.OriginIAM,
	}
	group5 = model.Group{
		UUID:       groupUUID5,
		TenantUUID: tenantUUID1,
		Groups:     []string{groupUUID2, groupUUID1},
		Origin:     model.OriginIAM,
	}
)

func createGroups(t *testing.T, repo *model.GroupRepository, groups ...model.Group) {
	for _, group := range groups {
		tmp := group
		tmp.FullIdentifier = uuid.New()
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func groupFixture(t *testing.T, store *io.MemoryStore) {
	gs := []model.Group{group2, group3, group4, group1, group5}
	for i := range gs {
		gs[i].Subjects = appendSubjects(makeSubjectNotations(model.UserType, gs[i].Users),
			makeSubjectNotations(model.ServiceAccountType, gs[i].ServiceAccounts),
			makeSubjectNotations(model.GroupType, gs[i].Groups))
	}
	tx := store.Txn(true)
	repo := model.NewGroupRepository(tx)
	createGroups(t, repo, gs...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_ListGroups(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	groups, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range groups {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{groupUUID1, groupUUID2, groupUUID4, groupUUID5}, ids)
}

func Test_GetByID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	group, err := repo.GetByID(groupUUID1)

	dieOnErr(t, err)
	group1.Subjects = appendSubjects(makeSubjectNotations(model.UserType, group1.Users),
		makeSubjectNotations(model.ServiceAccountType, group1.ServiceAccounts),
		makeSubjectNotations(model.GroupType, group1.Groups))
	group.Version = group1.Version
	group.FullIdentifier = group1.FullIdentifier
	checkDeepEqual(t, &group1, group)
}

func Test_findDirectParentGroupsByUserUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindDirectParentGroupsByUserUUID(tenantUUID1, userUUID3)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}, groupUUID2: {}, groupUUID4: {}}, ids)
}

func Test_findDirectParentGroupsByServiceAccountUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindDirectParentGroupsByServiceAccountUUID(tenantUUID1, serviceAccountUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}}, ids)
}

func Test_findDirectParentGroupsByGroupUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindDirectParentGroupsByGroupUUID(tenantUUID1, groupUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID5: {}}, ids)
}

func Test_FindAllParentGroupsForUserUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindAllParentGroupsForUserUUID(tenantUUID1, userUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID2: {}, groupUUID4: {}, groupUUID5: {}}, ids)
}

func Test_FindAllParentGroupsForServiceAccountUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindAllParentGroupsForServiceAccountUUID(tenantUUID1, serviceAccountUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}, groupUUID5: {}}, ids)
}
