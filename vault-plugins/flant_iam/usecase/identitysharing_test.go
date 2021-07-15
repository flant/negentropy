package usecase

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	shareUUID1 = "00000000-0000-0000-0000-010000000000"
	shareUUID2 = "00000000-0000-0000-0000-020000000000"
)

var (
	share1 = model.IdentitySharing{
		UUID:                  shareUUID1,
		SourceTenantUUID:      tenantUUID1,
		DestinationTenantUUID: tenantUUID2,
		Groups:                []string{groupUUID2},
	}
	share2 = model.IdentitySharing{
		UUID:                  shareUUID2,
		SourceTenantUUID:      tenantUUID2,
		DestinationTenantUUID: tenantUUID1,
		Groups:                []string{groupUUID3},
	}
)

func createIdentitySharings(t *testing.T, repo *model.IdentitySharingRepository, shares ...model.IdentitySharing) {
	for _, share := range shares {
		tmp := share
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func identitySharingFixture(t *testing.T, store *io.MemoryStore) {
	shs := []model.IdentitySharing{share1, share2}
	tx := store.Txn(true)
	repo := model.NewIdentitySharingRepository(tx)
	createIdentitySharings(t, repo, shs...)
	err := tx.Commit()
	dieOnErr(t, err)
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

	shares, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range shares {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{shareUUID1}, ids)
}

func Test_ListForDestinationTenant(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture,
		identitySharingFixture).Txn(true)
	repo := model.NewIdentitySharingRepository(tx)

	shares, err := repo.ListForDestinationTenant(tenantUUID1)

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range shares {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{shareUUID2}, ids)
}
