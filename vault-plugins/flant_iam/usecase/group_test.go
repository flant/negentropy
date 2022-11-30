package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
)

func Test_ListGroups(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture).Txn(true)
	repository := iam_repo.NewGroupRepository(tx)

	groups, err := repository.List(fixtures.TenantUUID1, false)

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
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture).Txn(true)
	repository := iam_repo.NewGroupRepository(tx)

	group, err := repository.GetByID(fixtures.GroupUUID1)

	require.NoError(t, err)
	group1 := fixtures.Groups()[1]
	group1.Members = appendMembers(makeMemberNotations(model.UserType, group1.Users),
		makeMemberNotations(model.ServiceAccountType, group1.ServiceAccounts),
		makeMemberNotations(model.GroupType, group1.Groups))
	group.Version = group1.Version
	group.FullIdentifier = group1.FullIdentifier
	require.Equal(t, &group1, group)
}

func Test_findDirectParentGroupsByUserUUID(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture).Txn(true)
	repository := iam_repo.NewGroupRepository(tx)

	ids, err := repository.FindDirectParentGroupsByUserUUID(fixtures.UserUUID3)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID1, fixtures.GroupUUID2, fixtures.GroupUUID3, fixtures.GroupUUID4}, stringSlice(ids))
}

func Test_findDirectParentGroupsByServiceAccountUUID(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture).Txn(true)
	repository := iam_repo.NewGroupRepository(tx)

	ids, err := repository.FindDirectParentGroupsByServiceAccountUUID(fixtures.ServiceAccountUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID1, fixtures.GroupUUID3}, stringSlice(ids))
}

func Test_findDirectParentGroupsByGroupUUID(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture).Txn(true)
	repository := iam_repo.NewGroupRepository(tx)

	ids, err := repository.FindDirectParentGroupsByGroupUUID(fixtures.GroupUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID5}, stringSlice(ids))
}

func Test_FindAllParentGroupsForUserUUID(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture).Txn(true)
	repository := iam_repo.NewGroupRepository(tx)

	ids, err := repository.FindAllParentGroupsForUserUUID(fixtures.UserUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.GroupUUID2, fixtures.GroupUUID4, fixtures.GroupUUID5}, stringSlice(ids))
}

func Test_FindAllParentGroupsForServiceAccountUUID(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture, ServiceAccountFixture, GroupFixture).Txn(true)
	repository := iam_repo.NewGroupRepository(tx)

	ids, err := repository.FindAllParentGroupsForServiceAccountUUID(fixtures.ServiceAccountUUID1)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		fixtures.GroupUUID1, fixtures.GroupUUID3,
		fixtures.GroupUUID4, fixtures.GroupUUID5,
	}, stringSlice(ids))
}
