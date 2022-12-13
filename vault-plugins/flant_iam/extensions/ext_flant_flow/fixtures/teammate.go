package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_uuid "github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	FlantUUID = "00000000-0000-4000-A000-000000000088"

	TeammateUUID1 = "00000000-0000-4000-A000-000000000101"
	TeammateUUID2 = "00000000-0000-4000-A000-000000000102"
	TeammateUUID3 = "00000000-0000-4000-A000-000000000103"
	TeammateUUID4 = "00000000-0000-4000-A000-000000000104"
	TeammateUUID5 = "00000000-0000-4000-A000-000000000105"
)

func Teammates() []model.FullTeammate {
	return []model.FullTeammate{
		{
			User: iam_model.User{
				UUID:           TeammateUUID1,
				Identifier:     "teammate1",
				FullIdentifier: "teammate1@flant",
				Email:          "teammate1@gmail.com",
				Origin:         "flant_flow",
				TenantUUID:     FlantUUID,
				Language:       "english",
			},
			TeamUUID:   TeamUUID1,
			RoleAtTeam: model.EngineerRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID2,
				Identifier:     "teammate2",
				FullIdentifier: "teammate2@flant",
				Email:          "teammate2@gmail.com",
				Origin:         "flant_flow",
				TenantUUID:     FlantUUID,
				Language:       "german",
			},
			TeamUUID:   TeamUUID2,
			RoleAtTeam: model.TeamLeadRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID3,
				Identifier:     "teammate3",
				FullIdentifier: "teammate3@flant",
				Email:          "teammate3@gmail.com",
				Origin:         "flant_flow",
				TenantUUID:     FlantUUID,
				Language:       "russian",
			},
			TeamUUID:   TeamUUID1,
			RoleAtTeam: model.ProjectManagerRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID4,
				Identifier:     "teammate4",
				FullIdentifier: "teammate4@flant",
				Email:          "teammate4@gmail.com",
				Origin:         "flant_flow",
				TenantUUID:     FlantUUID,
				Language:       "french",
			},
			TeamUUID:   TeamUUID3,
			RoleAtTeam: model.MemberRole,
		},
		{
			User: iam_model.User{
				UUID:           TeammateUUID5,
				Identifier:     "teammate5",
				FullIdentifier: "teammate5@flant",
				Email:          "teammate5@gmail.com",
				Origin:         "flant_flow",
				TenantUUID:     FlantUUID,
				Language:       "albanian",
			},
			TeamUUID:   TeamUUID3,
			RoleAtTeam: model.ManagerRole,
		},
	}
}

func RandomTeammateCreatePayload(team model.Team) map[string]interface{} {
	teammatesSet := Teammates()
	rand.Seed(time.Now().UnixNano())
	sample := teammatesSet[rand.Intn(len(teammatesSet))]

	sample.Identifier = "teammate_" + iam_uuid.New()
	sample.Email = fmt.Sprintf("%s@ex.com", RandomStr())
	sample.TeamUUID = team.UUID
	var role string
	for r := range model.TeamRoles[team.TeamType] {
		role = r
		break
	}
	sample.RoleAtTeam = role
	sample.GitlabAccount = iam_uuid.New() + "@gitlab.com"
	sample.GithubAccount = iam_uuid.New() + "@github.com"
	sample.TelegramAccount = "@telegram" + iam_uuid.New()
	sample.HabrAccount = iam_uuid.New() + "@habr.com"
	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	return payload
}
