package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createIdentitySharings(t *testing.T, repo *iam_repo.IdentitySharingRepository, shares ...model.IdentitySharing) {
	for _, share := range shares {
		tmp := share
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func identitySharingFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewIdentitySharingRepository(tx)
	createIdentitySharings(t, repo, fixtures.IdentitySharings()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ListIdentitySharing(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, userFixture, serviceAccountFixture, groupFixture,
		identitySharingFixture).Txn(true)
	repo := iam_repo.NewIdentitySharingRepository(tx)

	shares, err := repo.List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range shares {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.ShareUUID1, fixtures.ShareUUID3}, ids)
}

func Test_ListForDestinationTenant(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, userFixture, serviceAccountFixture, groupFixture,
		identitySharingFixture).Txn(true)
	repo := iam_repo.NewIdentitySharingRepository(tx)

	shares, err := repo.ListForDestinationTenant(fixtures.TenantUUID1)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range shares {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.ShareUUID2}, ids)
}

func Test_ListDestinationTenantsByGroupUUIDs(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, userFixture, serviceAccountFixture, groupFixture,
		identitySharingFixture).Txn(true)
	repo := iam_repo.NewIdentitySharingRepository(tx)

	shares, err := repo.ListDestinationTenantsByGroupUUIDs(fixtures.GroupUUID4, fixtures.GroupUUID3)

	require.NoError(t, err)
	ids := make([]string, 0)
	for tUUID := range shares {
		ids = append(ids, tUUID)
	}
	require.ElementsMatch(t, []string{fixtures.TenantUUID1, fixtures.TenantUUID2}, ids)
}

func Test_ListDestinationTenantsByGroupUUIDs2(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, userFixture, serviceAccountFixture, groupFixture,
		identitySharingFixture).Txn(true)
	repo := iam_repo.NewIdentitySharingRepository(tx)

	shares, err := repo.ListDestinationTenantsByGroupUUIDs(fixtures.GroupUUID1, fixtures.GroupUUID3)

	require.NoError(t, err)
	ids := make([]string, 0)
	for tUUID := range shares {
		ids = append(ids, tUUID)
	}
	require.ElementsMatch(t, []string{fixtures.TenantUUID1, fixtures.TenantUUID2}, ids)
}
