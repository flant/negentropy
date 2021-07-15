package usecase

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	serviceAccountUUID1 = "00000000-0000-0000-0000-000000000011"
	serviceAccountUUID2 = "00000000-0000-0000-0000-000000000012"
	serviceAccountUUID3 = "00000000-0000-0000-0000-000000000013"
	serviceAccountUUID4 = "00000000-0000-0000-0000-000000000014"
)

var (
	sa1 = model.ServiceAccount{
		UUID:           serviceAccountUUID1,
		TenantUUID:     tenantUUID1,
		Identifier:     "sa1",
		FullIdentifier: "sa1@test",
		Origin:         "test",
	}
	sa2 = model.ServiceAccount{
		UUID:           serviceAccountUUID2,
		TenantUUID:     tenantUUID1,
		Identifier:     "sa2",
		FullIdentifier: "sa2@test",
		Origin:         "test",
	}
	sa3 = model.ServiceAccount{
		UUID:           serviceAccountUUID3,
		TenantUUID:     tenantUUID1,
		Identifier:     "sa3",
		FullIdentifier: "sa3@test",
		Origin:         "test",
	}
	sa4 = model.ServiceAccount{
		UUID:           serviceAccountUUID4,
		TenantUUID:     tenantUUID2,
		Identifier:     "sa4",
		FullIdentifier: "sa4@test",
		Origin:         "test",
	}
)

func createServiceAccounts(t *testing.T, repo *model.ServiceAccountRepository, sas ...model.ServiceAccount) {
	for _, sa := range sas {
		tmp := sa
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func serviceAccountFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := model.NewServiceAccountRepository(tx)
	createServiceAccounts(t, repo, []model.ServiceAccount{sa1, sa2, sa3, sa4}...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_ServiceAccountList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, serviceAccountFixture).Txn(true)
	repo := model.NewServiceAccountRepository(tx)

	serviceAccounts, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range serviceAccounts {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{serviceAccountUUID1, serviceAccountUUID2, serviceAccountUUID3}, ids)
}
