package fixtures

import (
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

const (
	TenantUUID1 = "00000001-0000-0000-0000-000000000000"
	TenantUUID2 = "00000002-0000-0000-0000-000000000000"
)

func Clients() []model.Client {
	return []model.Client{
		{Tenant: iam_model.Tenant{
			UUID:       TenantUUID1,
			Identifier: "tenant1",
			Version:    "v1",
		}},
		{Tenant: iam_model.Tenant{
			UUID:       TenantUUID2,
			Identifier: "tenant2",
			Version:    "v1",
		}},
	}
}

func RandomClientCreatePayload() map[string]interface{} {
	clientSet := Clients()
	rand.Seed(time.Now().UnixNano())
	sample := clientSet[rand.Intn(len(clientSet))]
	return map[string]interface{}{
		"identifier": "Identifier_" + RandomStr(),
		"version":    sample.Version,
	}
}
