package fixtures

import (
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
)

const (
	TeamUUID1 = "00000101-0000-0000-0000-000000000000"
	TeamUUID2 = "00000102-0000-0000-0000-000000000000"
	TeamUUID3 = "00000103-0000-0000-0000-000000000000"
)

func Teams() []model.Team {
	return []model.Team{
		{
			UUID:           TeamUUID1,
			Version:        "v1",
			Identifier:     "team1",
			TeamType:       model.DevopsTeam,
			ParentTeamUUID: "",
		},
		{
			UUID:           TeamUUID2,
			Version:        "v1",
			Identifier:     "team2",
			TeamType:       model.DevopsTeam,
			ParentTeamUUID: "",
		},
		{
			UUID:           TeamUUID3,
			Version:        "v1",
			Identifier:     "team3",
			TeamType:       model.StandardTeam,
			ParentTeamUUID: "",
		},
	}
}

func TeamCreatePayload(sample model.Team) map[string]interface{} {
	return map[string]interface{}{
		"uuid":       sample.UUID,
		"identifier": "Identifier_" + RandomStr(),
		"version":    sample.Version,
		"team_type":  sample.TeamType,
		// "parent_team_uuid": sample.ParentTeamUUID,
	}
}

func RandomTeamCreatePayload() map[string]interface{} {
	teamSet := Teams()
	rand.Seed(time.Now().UnixNano())
	sample := teamSet[rand.Intn(len(teamSet))]
	return TeamCreatePayload(sample)
}
