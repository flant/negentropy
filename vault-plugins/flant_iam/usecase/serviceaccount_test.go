package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createServiceAccounts(t *testing.T, repo *model.ServiceAccountRepository, sas ...model.ServiceAccount) {
	for _, sa := range sas {
		tmp := sa
		tmp.FullIdentifier = uuid.New() // delete after bringing full identifiers to usecases
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func serviceAccountFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := model.NewServiceAccountRepository(tx)
	createServiceAccounts(t, repo, fixtures.ServiceAccounts()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ServiceAccountList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, serviceAccountFixture).Txn(true)
	repo := model.NewServiceAccountRepository(tx)

	serviceAccounts, err := repo.List(fixtures.TenantUUID1)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range serviceAccounts {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.ServiceAccountUUID1, fixtures.ServiceAccountUUID2, fixtures.ServiceAccountUUID3}, ids)
}
