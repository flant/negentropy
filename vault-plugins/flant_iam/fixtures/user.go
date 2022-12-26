package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	UserUUID1 = "00000000-0000-4000-A000-000000000001"
	UserUUID2 = "00000000-0000-4000-A000-000000000002"
	UserUUID3 = "00000000-0000-4000-A000-000000000003"
	UserUUID4 = "00000000-0000-4000-A000-000000000004"
	UserUUID5 = "00000000-0000-4000-A000-000000000005"
)

func Users() []model.User {
	return []model.User{
		{
			UUID:           UserUUID1,
			TenantUUID:     TenantUUID1,
			Identifier:     "user1",
			FullIdentifier: "user1@test",
			Email:          "user1@gmail.com",
			Origin:         "test",
			Language:       "english",
		},
		{
			UUID:           UserUUID2,
			TenantUUID:     TenantUUID1,
			Identifier:     "user2",
			FullIdentifier: "user2@test",
			Email:          "user2@gmail.com",
			Origin:         "test",
			Language:       "german",
		},
		{
			UUID:           UserUUID3,
			TenantUUID:     TenantUUID1,
			Identifier:     "user3",
			FullIdentifier: "user3@test",
			Email:          "user3@gmail.com",
			Origin:         "test",
			Language:       "russian",
		},
		{
			UUID:           UserUUID4,
			TenantUUID:     TenantUUID1,
			Identifier:     "user4",
			FullIdentifier: "user4@test",
			Email:          "user4@gmail.com",
			Origin:         "test",
			Language:       "french",
		},
		{
			UUID:           UserUUID5,
			TenantUUID:     TenantUUID2,
			Identifier:     "user5",
			FullIdentifier: "user5@test",
			Email:          "user5@gmail.com",
			Origin:         "test",
			Language:       "albanian",
		},
	}
}

func RandomUserCreatePayload() map[string]interface{} {
	userSet := Users()
	rand.Seed(time.Now().UnixNano())
	sample := userSet[rand.Intn(len(userSet))]

	sample.Identifier = uuid.New()
	sample.Email = fmt.Sprintf("%s@ex.com", RandomStr())

	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	return payload
}
