package fixtures

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const (
	ShareUUID1 = "00000000-0000-0000-0000-010000000000"
	ShareUUID2 = "00000000-0000-0000-0000-020000000000"
)

func IdentitySharings() []model.IdentitySharing {
	return []model.IdentitySharing{
		{
			UUID:                  ShareUUID1,
			SourceTenantUUID:      TenantUUID1,
			DestinationTenantUUID: TenantUUID2,
			Groups:                []string{GroupUUID2},
		},
		{
			UUID:                  ShareUUID2,
			SourceTenantUUID:      TenantUUID2,
			DestinationTenantUUID: TenantUUID1,
			Groups:                []string{GroupUUID3},
		},
	}
}
