package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createIdentitySharings(t *testing.T, repo *model.IdentitySharingRepository, shares ...model.IdentitySharing) {
	for _, share := range shares {
		tmp := share
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func identitySharingFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := model.NewIdentitySharingRepository(tx)
	createIdentitySharings(t, repo, fixtures.IdentitySharings()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_IdentitySharingDbSchema(t *testing.T) {
	schema := model.IdentitySharingSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("identity sharing schema is invalid: %v", err)
	}
}

func Test_ListIdentitySharing(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture,
		identitySharingFixture).Txn(true)
	repo := model.NewIdentitySharingRepository(tx)

	shares, err := repo.List(fixtures.TenantUUID1)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range shares {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.ShareUUID1}, ids)
}

func Test_ListForDestinationTenant(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture,
		identitySharingFixture).Txn(true)
	repo := model.NewIdentitySharingRepository(tx)

	shares, err := repo.ListForDestinationTenant(fixtures.TenantUUID1)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range shares {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.ShareUUID2}, ids)
}
