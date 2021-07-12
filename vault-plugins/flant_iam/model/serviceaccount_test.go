package model

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	serviceAccountUUID1 = "00000000-0003-0000-0000-000000000011"
	serviceAccountUUID2 = "00000000-0003-0000-0000-000000000012"
	serviceAccountUUID3 = "00000000-0003-0000-0000-000000000013"
	serviceAccountUUID4 = "00000000-0003-0000-0000-000000000014"
)

var (
	sa1 = ServiceAccount{
		UUID:           serviceAccountUUID1,
		TenantUUID:     tenantUUID1,
		Identifier:     "sa1",
		FullIdentifier: "sa1@test",
		Origin:         "test",
	}
	sa2 = ServiceAccount{
		UUID:           serviceAccountUUID2,
		TenantUUID:     tenantUUID1,
		Identifier:     "sa2",
		FullIdentifier: "sa2@test",
		Origin:         "test",
	}
	sa3 = ServiceAccount{
		UUID:           serviceAccountUUID3,
		TenantUUID:     tenantUUID1,
		Identifier:     "sa3",
		FullIdentifier: "sa3@test",
		Origin:         "test",
	}
	sa4 = ServiceAccount{
		UUID:           serviceAccountUUID4,
		TenantUUID:     tenantUUID2,
		Identifier:     "sa4",
		FullIdentifier: "sa4@test",
		Origin:         "test",
	}
)

func createServiceAccounts(t *testing.T, repo *ServiceAccountRepository, sas ...ServiceAccount) {
	for _, sa := range sas {
		tmp := sa
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func serviceAccountFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := NewServiceAccountRepository(tx)
	createServiceAccounts(t, repo, []ServiceAccount{sa1, sa2, sa3, sa4}...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_ServiceAccountDbSchema(t *testing.T) {
	schema := ServiceAccountSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("service account schema is invalid: %v", err)
	}
}

func Test_ServiceAccountList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, serviceAccountFixture).Txn(true)
	repo := NewServiceAccountRepository(tx)

	serviceAccounts, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range serviceAccounts {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{serviceAccountUUID1, serviceAccountUUID2, serviceAccountUUID3}, ids)
}
