package fixtures

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

const (
	ServiceAccountUUID1 = "00000000-0000-0000-0000-000000000011"
	ServiceAccountUUID2 = "00000000-0000-0000-0000-000000000012"
	ServiceAccountUUID3 = "00000000-0000-0000-0000-000000000013"
	ServiceAccountUUID4 = "00000000-0000-0000-0000-000000000014"
)

func ServiceAccounts() []model.ServiceAccount {
	return []model.ServiceAccount{
		{
			UUID:           ServiceAccountUUID1,
			TenantUUID:     TenantUUID1,
			Identifier:     "sa1",
			FullIdentifier: "sa1@test",
			Origin:         "test",
			Version:        RandomStr(),
			CIDRs:          []string{"0.0.0.0/0"},
			TokenTTL:       100 * time.Second,
			TokenMaxTTL:    1000 * time.Second,
		},
		{
			UUID:           ServiceAccountUUID2,
			TenantUUID:     TenantUUID1,
			Identifier:     "sa2",
			FullIdentifier: "sa2@test",
			Origin:         "test",
			Version:        RandomStr(),
			CIDRs:          []string{"0.0.0.0/0"},
			TokenTTL:       100 * time.Second,
			TokenMaxTTL:    1000 * time.Second,
		},
		{
			UUID:           ServiceAccountUUID3,
			TenantUUID:     TenantUUID1,
			Identifier:     "sa3",
			FullIdentifier: "sa3@test",
			Origin:         "test",
			Version:        RandomStr(),
			CIDRs:          []string{"0.0.0.0/0"},
			TokenTTL:       100 * time.Second,
			TokenMaxTTL:    1000 * time.Second,
		},
		{
			UUID:           ServiceAccountUUID4,
			TenantUUID:     TenantUUID2,
			Identifier:     "sa4",
			FullIdentifier: "sa4@test",
			Origin:         "test",
			Version:        RandomStr(),
			CIDRs:          []string{"0.0.0.0/0"},
			TokenTTL:       100 * time.Second,
			TokenMaxTTL:    1000 * time.Second,
		},
	}
}

func RandomServiceAccountCreatePayload() map[string]interface{} {
	saSet := ServiceAccounts()
	rand.Seed(time.Now().UnixNano())
	sample := saSet[rand.Intn(len(saSet))]

	sample.Identifier = "Identifier_" + RandomStr()

	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	return payload
}
