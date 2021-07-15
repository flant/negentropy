package usecase

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	tenantUUID1 = "00000001-0000-0000-0000-000000000000"
	tenantUUID2 = "00000002-0000-0000-0000-000000000000"
)



var (
	tenant1 = model.Tenant{
		UUID:         tenantUUID1,
		Identifier:   "tenant1",
		Version:      "v1",
		FeatureFlags: nil,
	}

	tenant2 = model.Tenant{
		UUID:         tenantUUID2,
		Identifier:   "tenant2",
		Version:      "v1",
		FeatureFlags: nil,
	}
)

func createTenants(t *testing.T, repo *model.TenantRepository, tenants ...model.Tenant) {
	for _, tenant := range tenants {
		tmp := tenant
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func tenantFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := model.NewTenantRepository(tx)
	createTenants(t, repo, []model.Tenant{tenant1, tenant2}...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_TenantList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture).Txn(true)
	repo := model.NewTenantRepository(tx)

	tenants, err := repo.List()

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range tenants {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{tenantUUID1, tenantUUID2}, ids)
}
