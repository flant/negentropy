package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createGroups(t *testing.T, repo *model.GroupRepository, groups ...model.Group) {
	for _, group := range groups {
		tmp := group
		tmp.FullIdentifier = uuid.New()
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func groupFixture(t *testing.T, store *io.MemoryStore) {
	gs := fixtures.Groups()
	for i := range gs {
		gs[i].Members = appendMembers(makeMemberNotations(model.UserType, gs[i].Users),
			makeMemberNotations(model.ServiceAccountType, gs[i].ServiceAccounts),
			makeMemberNotations(model.GroupType, gs[i].Groups))
	}
	tx := store.Txn(true)
	repo := model.NewGroupRepository(tx)
	createGroups(t, repo, gs...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ListGroups(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	groups, err := repo.List(fixtures.TenantUUID1)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range groups {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{
		fixtures.GroupUUID1, fixtures.GroupUUID2,
		fixtures.GroupUUID4, fixtures.GroupUUID5,
	}, ids)
}

func Test_GetByID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	group, err := repo.GetByID(fixtures.GroupUUID1)

	require.NoError(t, err)
	group1 := fixtures.Groups()[0]
	group1.Members = appendMembers(makeMemberNotations(model.UserType, group1.Users),
		makeMemberNotations(model.ServiceAccountType, group1.ServiceAccounts),
		makeMemberNotations(model.GroupType, group1.Groups))
	group.Version = group1.Version
	group.FullIdentifier = group1.FullIdentifier
	require.Equal(t, &group1, group)
}

func Test_findDirectParentGroupsByUserUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindDirectParentGroupsByUserUUID(fixtures.TenantUUID1, fixtures.UserUUID3)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID1, fixtures.GroupUUID2, fixtures.GroupUUID4}, stringSlice(ids))
}

func Test_findDirectParentGroupsByServiceAccountUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindDirectParentGroupsByServiceAccountUUID(fixtures.TenantUUID1, fixtures.ServiceAccountUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID1}, stringSlice(ids))
}

func Test_findDirectParentGroupsByGroupUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindDirectParentGroupsByGroupUUID(fixtures.TenantUUID1, fixtures.GroupUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID5}, stringSlice(ids))
}

func Test_FindAllParentGroupsForUserUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindAllParentGroupsForUserUUID(fixtures.TenantUUID1, fixtures.UserUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID2, fixtures.GroupUUID4, fixtures.GroupUUID5}, stringSlice(ids))
}

func Test_FindAllParentGroupsForServiceAccountUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := model.NewGroupRepository(tx)

	ids, err := repo.FindAllParentGroupsForServiceAccountUUID(fixtures.TenantUUID1, fixtures.ServiceAccountUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID1, fixtures.GroupUUID5}, stringSlice(ids))
}
