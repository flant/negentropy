package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createTenants(t *testing.T, repo *model.TenantRepository, tenants ...model.Tenant) {
	for _, tenant := range tenants {
		tmp := tenant
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func tenantFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := model.NewTenantRepository(tx)
	createTenants(t, repo, fixtures.Tenants()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_TenantList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture).Txn(true)
	repo := model.NewTenantRepository(tx)

	tenants, err := repo.List(false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range tenants {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.TenantUUID1, fixtures.TenantUUID2}, ids)
}
