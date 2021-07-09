package model

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	tenantUUID1 = "00000001-0000-0000-0000-000000000000"
	tenantUUID2 = "00000002-0000-0000-0000-000000000000"
)

var (
	tenant1 = Tenant{
		UUID:         tenantUUID1,
		Identifier:   "tenant1",
		Version:      "v1",
		FeatureFlags: nil,
	}

	tenant2 = Tenant{
		UUID:         tenantUUID2,
		Identifier:   "tenant2",
		FeatureFlags: nil,
	}
)

func createTenants(t *testing.T, repo *TenantRepository, tenants ...Tenant) {
	for _, tenant := range tenants {
		tmp := tenant
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func Test_TenantMarshalling(t *testing.T) {
	ten := &Tenant{
		UUID:       uuid.New(),
		Identifier: "somefun",
	}

	raw, err := json.Marshal(ten)
	if err != nil {
		t.Fatalf("cannot marshal tenant with sensitive data: %v", err)
	}

	ten2 := &Tenant{}
	err = json.Unmarshal(raw, &ten2)
	if err != nil {
		t.Fatalf("cannot unmarshal tenant back: %v", err)
	}

	if !reflect.DeepEqual(ten, ten2) {
		t.Fatalf("tenant changed during marshalling/unmarshalling: was=%v, became=%v", ten, ten2)
	}
}

func Test_TenantDbSchema(t *testing.T) {
	schema := TenantSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("tenant schema is invalid: %v", err)
	}
}

func Test_TenantList(t *testing.T) {
	schema := TenantSchema()
	store, err := io.NewMemoryStore(schema, nil)
	dieOnErr(t, err)
	tx := store.Txn(true)
	defer tx.Abort()
	repo := NewTenantRepository(tx)
	createTenants(t, repo, []Tenant{tenant1, tenant2}...)

	ids, err := repo.List()
	dieOnErr(t, err)
	checkDeepEqual(t, []string{tenantUUID1, tenantUUID2}, ids)
}
