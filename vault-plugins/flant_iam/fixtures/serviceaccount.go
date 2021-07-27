package fixtures

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

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
		},
		{
			UUID:           ServiceAccountUUID2,
			TenantUUID:     TenantUUID1,
			Identifier:     "sa2",
			FullIdentifier: "sa2@test",
			Origin:         "test",
		},
		{
			UUID:           ServiceAccountUUID3,
			TenantUUID:     TenantUUID1,
			Identifier:     "sa3",
			FullIdentifier: "sa3@test",
			Origin:         "test",
		},
		{
			UUID:           ServiceAccountUUID4,
			TenantUUID:     TenantUUID2,
			Identifier:     "sa4",
			FullIdentifier: "sa4@test",
			Origin:         "test",
		},
	}
}
