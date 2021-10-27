package fixtures

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	GroupUUID1 = "00000000-0001-0000-0000-000000000000"
	GroupUUID2 = "00000000-0002-0000-0000-000000000000"
	GroupUUID3 = "00000000-0003-0000-0000-000000000000"
	GroupUUID4 = "00000000-0004-0000-0000-000000000000"
	GroupUUID5 = "00000000-0005-0000-0000-000000000000"
)

func Groups() []model.Group {
	return []model.Group{
		{
			UUID:            GroupUUID1,
			TenantUUID:      TenantUUID1,
			Identifier:      "group1",
			Users:           []string{UserUUID2, UserUUID3},
			Groups:          []string{GroupUUID3},
			ServiceAccounts: []string{ServiceAccountUUID1},
			Origin:          model.OriginIAM,
		},
		{
			UUID:       GroupUUID2,
			TenantUUID: TenantUUID1,
			Identifier: "group2",
			Users:      []string{UserUUID1, UserUUID3},
			Origin:     model.OriginIAM,
		},
		{
			UUID:            GroupUUID3,
			TenantUUID:      TenantUUID2,
			Identifier:      "group3",
			Users:           []string{UserUUID3, UserUUID4},
			ServiceAccounts: []string{ServiceAccountUUID1},
			Origin:          model.OriginIAM,
		},
		{
			UUID:            GroupUUID4,
			TenantUUID:      TenantUUID1,
			Identifier:      "group4",
			Users:           []string{UserUUID2, UserUUID3},
			Groups:          []string{GroupUUID2, GroupUUID3},
			ServiceAccounts: []string{ServiceAccountUUID2, ServiceAccountUUID3},
			Origin:          model.OriginIAM,
		},
		{
			UUID:       GroupUUID5,
			TenantUUID: TenantUUID1,
			Identifier: "group5",
			Groups:     []string{GroupUUID2, GroupUUID1},
			Origin:     model.OriginIAM,
		},
	}
}

func RandomGroupCreatePayload() map[string]interface{} {
	groupSet := Groups()
	rand.Seed(time.Now().UnixNano())
	sample := groupSet[rand.Intn(len(groupSet))]

	sample.UUID = ""
	sample.Identifier = uuid.New()
	sample.Groups = nil
	sample.ServiceAccounts = nil
	sample.Members = nil
	sample.Users = nil

	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	return payload
}
