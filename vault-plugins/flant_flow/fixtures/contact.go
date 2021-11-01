package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	UserUUID1 = "00000000-0000-0000-0000-000000000001"
	UserUUID2 = "00000000-0000-0000-0000-000000000002"
	UserUUID3 = "00000000-0000-0000-0000-000000000003"
	UserUUID4 = "00000000-0000-0000-0000-000000000004"
	UserUUID5 = "00000000-0000-0000-0000-000000000005"
)

func Contacts() []model.Contact {
	return []model.Contact{
		{
			User: iam_model.User{
				UUID:           UserUUID1,
				TenantUUID:     TenantUUID1,
				Identifier:     "user1",
				FullIdentifier: "user1@test",
				Email:          "user1@mail.com",
				Origin:         "test",
			},
			Credentials: map[iam_model.ProjectUUID]model.ContactRole{
				ProjectUUID1: model.RegularContact,
				ProjectUUID2: model.AuthorizedContact,
				ProjectUUID3: model.Representative,
				ProjectUUID4: model.Plenipotentiary,
			},
		},
		{
			User: iam_model.User{
				UUID:           UserUUID2,
				TenantUUID:     TenantUUID1,
				Identifier:     "user2",
				FullIdentifier: "user2@test",
				Email:          "user2@mail.com",
				Origin:         "test",
			},
		},
		{
			User: iam_model.User{
				UUID:           UserUUID3,
				TenantUUID:     TenantUUID1,
				Identifier:     "user3",
				FullIdentifier: "user3@test",
				Email:          "user3@mail.com",
				Origin:         "test",
			},
		},
		{
			User: iam_model.User{
				UUID:           UserUUID4,
				TenantUUID:     TenantUUID1,
				Identifier:     "user4",
				FullIdentifier: "user4@test",
				Email:          "user4@mail.com",
				Origin:         "test",
			},
		},
		{
			User: iam_model.User{
				UUID:           UserUUID5,
				TenantUUID:     TenantUUID2,
				Identifier:     "user4",
				FullIdentifier: "user4@test",
				Email:          "user4@mail.com",
				Origin:         "test",
			},
		},
	}
}

func RandomContactCreatePayload() map[string]interface{} {
	contactSet := Contacts()
	rand.Seed(time.Now().UnixNano())
	sample := contactSet[rand.Intn(len(contactSet))]

	sample.Identifier = uuid.New()
	sample.Email = fmt.Sprintf("%s@ex.com", RandomStr())
	sample.Credentials = Contacts()[0].Credentials

	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	return payload
}
