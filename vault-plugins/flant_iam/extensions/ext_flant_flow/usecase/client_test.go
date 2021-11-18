package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createClients(t *testing.T, repo *repo.ClientRepository, tenants ...model.Client) {
	for _, tenant := range tenants {
		tmp := tenant
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func clientFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := repo.NewClientRepository(tx)
	createClients(t, repo, fixtures.Clients()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ClientList(t *testing.T) {
	tx := runFixtures(t, clientFixture).Txn(true)
	repo := repo.NewClientRepository(tx)

	clients, err := repo.List(false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range clients {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.TenantUUID1, fixtures.TenantUUID2}, ids)
}
