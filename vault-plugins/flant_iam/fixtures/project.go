package fixtures

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const (
	ProjectUUID1 = "00000000-0100-0000-0000-000000000000"
	ProjectUUID2 = "00000000-0200-0000-0000-000000000000"
	ProjectUUID3 = "00000000-0300-0000-0000-000000000000"
	ProjectUUID4 = "00000000-0400-0000-0000-000000000000"
	ProjectUUID5 = "00000000-0500-0000-0000-000000000000"
)

func Projects() []model.Project {
	return []model.Project{
		{
			UUID:       ProjectUUID1,
			TenantUUID: TenantUUID1,
			Identifier: "pr1",
		},
		{
			UUID:       ProjectUUID2,
			TenantUUID: TenantUUID1,
			Identifier: "pr2",
		},
		{
			UUID:       ProjectUUID3,
			TenantUUID: TenantUUID1,
			Identifier: "pr3",
		},
		{
			UUID:       ProjectUUID4,
			TenantUUID: TenantUUID1,
			Identifier: "pr4",
		},
		{
			UUID:       ProjectUUID5,
			TenantUUID: TenantUUID2,
			Identifier: "pr5",
		},
	}
}