package fixtures

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const (
	TenantUUID1 = "00000001-0000-0000-0000-000000000000"
	TenantUUID2 = "00000002-0000-0000-0000-000000000000"
)

func Tenants() []model.Tenant {
	return []model.Tenant{
		{
			UUID:         TenantUUID1,
			Identifier:   "tenant1",
			Version:      "v1",
			FeatureFlags: nil,
		},
		{
			UUID:         TenantUUID2,
			Identifier:   "tenant2",
			Version:      "v1",
			FeatureFlags: nil,
		},
	}
}
