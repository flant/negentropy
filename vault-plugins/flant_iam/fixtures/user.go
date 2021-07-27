package fixtures

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const (
	UserUUID1 = "00000000-0000-0000-0000-000000000001"
	UserUUID2 = "00000000-0000-0000-0000-000000000002"
	UserUUID3 = "00000000-0000-0000-0000-000000000003"
	UserUUID4 = "00000000-0000-0000-0000-000000000004"
	UserUUID5 = "00000000-0000-0000-0000-000000000005"
)

func Users() []model.User {
	return []model.User{
		{
			UUID:           UserUUID1,
			TenantUUID:     TenantUUID1,
			Identifier:     "user1",
			FullIdentifier: "user1@test",
			Email:          "user1@mail.com",
			Origin:         "test",
		},
		{
			UUID:           UserUUID2,
			TenantUUID:     TenantUUID1,
			Identifier:     "user2",
			FullIdentifier: "user2@test",
			Email:          "user2@mail.com",
			Origin:         "test",
		},
		{
			UUID:           UserUUID3,
			TenantUUID:     TenantUUID1,
			Identifier:     "user3",
			FullIdentifier: "user3@test",
			Email:          "user3@mail.com",
			Origin:         "test",
		},
		{
			UUID:           UserUUID4,
			TenantUUID:     TenantUUID1,
			Identifier:     "user4",
			FullIdentifier: "user4@test",
			Email:          "user4@mail.com",
			Origin:         "test",
		},
		{
			UUID:           UserUUID5,
			TenantUUID:     TenantUUID2,
			Identifier:     "user4",
			FullIdentifier: "user4@test",
			Email:          "user4@mail.com",
			Origin:         "test",
		},
	}
}
