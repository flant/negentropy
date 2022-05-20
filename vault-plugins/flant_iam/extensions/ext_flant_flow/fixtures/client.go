package fixtures

import (
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
)

const (
	TenantUUID1 = "00000001-0000-0000-0000-000000000000"
	TenantUUID2 = "00000002-0000-0000-0000-000000000000"
)

func Clients() []model.Client {
	return []model.Client{
		{
			UUID:       TenantUUID1,
			Identifier: "tenant1",
			Version:    "v1",
			Language:   "english",
		},
		{
			UUID:       TenantUUID2,
			Identifier: "tenant2",
			Version:    "v1",
			Language:   "russian",
		},
	}
}

func RandomClientCreatePayload() map[string]interface{} {
	clientSet := Clients()
	rand.Seed(time.Now().UnixNano())
	sample := clientSet[rand.Intn(len(clientSet))]
	return map[string]interface{}{
		"identifier":       "Client_" + RandomStr(),
		"resource_version": sample.Version,
		"language":         sample.Language,
	}
}
