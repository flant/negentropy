package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_uuid "github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

const (
	TeammateUUID1 = "00000000-0000-0000-0000-000000000001"
	TeammateUUID2 = "00000000-0000-0000-0000-000000000002"
	TeammateUUID3 = "00000000-0000-0000-0000-000000000003"
	TeammateUUID4 = "00000000-0000-0000-0000-000000000004"
	TeammateUUID5 = "00000000-0000-0000-0000-000000000005"
)

func Teammates() []model.Teammate {
	return []model.Teammate{
		{
			User: iam_model.User{
				UUID:           TeammateUUID1,
				Identifier:     "user1",
				FullIdentifier: "user1@test",
				Email:          "user1@mail.com",
				Origin:         "test",
			},
			TeamUUID:   TenantUUID1,
			RoleAtTeam: model.EngineerRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID2,
				Identifier:     "user2",
				FullIdentifier: "user2@test",
				Email:          "user2@mail.com",
				Origin:         "test",
			},
			TeamUUID:   TeamUUID2,
			RoleAtTeam: model.TeamLeadRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID3,
				Identifier:     "user3",
				FullIdentifier: "user3@test",
				Email:          "user3@mail.com",
				Origin:         "test",
			},
			TeamUUID:   TeamUUID1,
			RoleAtTeam: model.ProjectManagerRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID4,
				Identifier:     "user4",
				FullIdentifier: "user4@test",
				Email:          "user4@mail.com",
				Origin:         "test",
			},
			TeamUUID:   TeamUUID3,
			RoleAtTeam: model.MemberRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID5,
				Identifier:     "user4",
				FullIdentifier: "user4@test",
				Email:          "user4@mail.com",
				Origin:         "test",
			},
			TeamUUID:   TeamUUID3,
			RoleAtTeam: model.ManagerRole,
		},
	}
}

func RandomTeammateCreatePayload() map[string]interface{} {
	teammatesSet := Teammates()
	rand.Seed(time.Now().UnixNano())
	sample := teammatesSet[rand.Intn(len(teammatesSet))]

	sample.Identifier = iam_uuid.New()
	sample.Email = fmt.Sprintf("%s@ex.com", RandomStr())

	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	return payload
}
