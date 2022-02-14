package fixtures

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

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
			Project: iam_model.Project{
				UUID:       ProjectUUID1,
				TenantUUID: TenantUUID1,
				Identifier: "pr1",
			},
			ServicePacks: map[model.ServicePackName]model.ServicePackCFG{
				model.L1: model.L1ServicePackCFG{},
			},
		},
		{
			Project: iam_model.Project{
				UUID:       ProjectUUID2,
				TenantUUID: TenantUUID1,
				Identifier: "pr2",
			},
			ServicePacks: map[model.ServicePackName]model.ServicePackCFG{
				model.L1: model.L1ServicePackCFG{},
			},
		},
		{
			Project: iam_model.Project{
				UUID:       ProjectUUID3,
				TenantUUID: TenantUUID1,
				Identifier: "pr3",
			},
			ServicePacks: map[model.ServicePackName]model.ServicePackCFG{
				model.Mk8s: nil,
			},
		},
		{
			Project: iam_model.Project{
				UUID:       ProjectUUID4,
				TenantUUID: TenantUUID1,
				Identifier: "pr4",
			},
			ServicePacks: map[model.ServicePackName]model.ServicePackCFG{
				model.Deckhouse: nil,
			},
		},
		{
			Project: iam_model.Project{
				UUID:       ProjectUUID5,
				TenantUUID: TenantUUID2,
				Identifier: "pr5",
			},
			ServicePacks: map[model.ServicePackName]model.ServicePackCFG{
				model.Consulting: nil,
			},
		},
	}
}

func ProjectCreatePayload(sample model.Project) map[string]interface{} {
	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	sps := []string{}
	for k := range sample.ServicePacks {
		if k != model.DevOps {
			sps = append(sps, k)
		}
	}
	payload["service_packs"] = sps
	return payload
}

func RandomProjectCreatePayload() map[string]interface{} {
	projectSet := Projects()
	rand.Seed(time.Now().UnixNano())
	sample := projectSet[rand.Intn(len(projectSet))]

	sample.Identifier = uuid.New()
	sample.UUID = ""

	return ProjectCreatePayload(sample)
}
