package fixtures

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

const (
	RbUUID1 = "00000000-0000-0001-0000-000000000000"
	RbUUID2 = "00000000-0000-0002-0000-000000000000"
	RbUUID3 = "00000000-0000-0003-0000-000000000000"
	RbUUID4 = "00000000-0000-0004-0000-000000000000"
	RbUUID5 = "00000000-0000-0005-0000-000000000000"
	// tenant_scoped_roles
	RbUUID6 = "00000000-0000-0006-0000-000000000000"
	RbUUID7 = "00000000-0000-0007-0000-000000000000"
	RbUUID8 = "00000000-0000-0008-0000-000000000000"
)

func RoleBindings() []model.RoleBinding {
	return []model.RoleBinding{
		{
			UUID:            RbUUID1,
			TenantUUID:      TenantUUID1,
			Description:     "rb1",
			ValidTill:       100,
			RequireMFA:      false,
			Users:           []string{UserUUID1, UserUUID2},
			Groups:          []string{GroupUUID2, GroupUUID3},
			ServiceAccounts: []string{ServiceAccountUUID1},
			AnyProject:      false,
			Projects:        []model.ProjectUUID{ProjectUUID1, ProjectUUID3},
			Roles: []model.BoundRole{{
				Name:    RoleName1,
				Options: map[string]interface{}{"o1": "data1"},
			}},
			Origin: consts.OriginIAM,
		},
		{
			UUID:        RbUUID2,
			TenantUUID:  TenantUUID2,
			Description: "rb2",
			ValidTill:   110,
			RequireMFA:  false,
			Users:       []string{UserUUID1, UserUUID2},
			AnyProject:  true,
			Projects:    nil,
			Roles: []model.BoundRole{{
				Name:    RoleName1,
				Options: map[string]interface{}{"o1": "data2"},
			}},
			Origin: consts.OriginIAM,
		},
		{
			UUID:            RbUUID3,
			TenantUUID:      TenantUUID1,
			Description:     "rb3",
			ValidTill:       120,
			RequireMFA:      false,
			Users:           []string{UserUUID2},
			Groups:          []string{GroupUUID2, GroupUUID5},
			ServiceAccounts: []string{ServiceAccountUUID2},
			AnyProject:      true,
			Projects:        nil,
			Roles: []model.BoundRole{{
				Name:    RoleName5,
				Options: map[string]interface{}{"o1": "data3"},
			}, {
				Name:    RoleName7,
				Options: map[string]interface{}{"o1": "data4"},
			}},
			Origin: consts.OriginIAM,
		},
		{
			UUID:        RbUUID4,
			TenantUUID:  TenantUUID1,
			Description: "rb4",
			ValidTill:   150,
			RequireMFA:  false,
			Users:       []string{UserUUID1},
			AnyProject:  false,
			Projects:    []model.ProjectUUID{ProjectUUID3, ProjectUUID4},
			Roles: []model.BoundRole{{
				Name:    RoleName8,
				Options: map[string]interface{}{"o1": "data5"},
			}},
			Origin: consts.OriginIAM,
		},
		{
			UUID:            RbUUID5,
			TenantUUID:      TenantUUID1,
			Description:     "rb5",
			ValidTill:       160,
			RequireMFA:      false,
			ServiceAccounts: []string{ServiceAccountUUID1},
			AnyProject:      false,
			Projects:        []model.ProjectUUID{ProjectUUID3, ProjectUUID1},
			Roles: []model.BoundRole{{
				Name:    RoleName1,
				Options: map[string]interface{}{"o1": "data6"},
			}},
			Origin: consts.OriginIAM,
		},
		{
			UUID:            RbUUID6,
			TenantUUID:      TenantUUID1,
			Description:     "rb6",
			ValidTill:       170,
			RequireMFA:      false,
			ServiceAccounts: []string{ServiceAccountUUID2},
			AnyProject:      false,
			Projects:        nil,
			Roles: []model.BoundRole{{
				Name:    RoleName9,
				Options: map[string]interface{}{"o1": "data7"},
			}},
			Origin: consts.OriginIAM,
		},
		{
			UUID:        RbUUID7,
			TenantUUID:  TenantUUID1,
			Description: "rb7",
			ValidTill:   180,
			RequireMFA:  false,
			Groups:      []model.GroupUUID{GroupUUID4},
			AnyProject:  false,
			Projects:    nil,
			Roles: []model.BoundRole{{
				Name:    RoleName10,
				Options: map[string]interface{}{"o1": "data8"},
			}},
			Origin: consts.OriginIAM,
		},
		{
			UUID:        RbUUID8,
			TenantUUID:  TenantUUID1,
			Description: "rb8",
			ValidTill:   190,
			RequireMFA:  false,
			Users:       []model.UserUUID{UserUUID2},
			AnyProject:  false,
			Projects:    nil,
			Roles: []model.BoundRole{{
				Name:    RoleName9,
				Options: map[string]interface{}{"o1": "data9"},
			}},
			Origin: consts.OriginIAM,
		},
	}
}

func RandomRoleBindingCreatePayload() map[string]interface{} {
	rbSet := RoleBindings()
	rand.Seed(time.Now().UnixNano())
	sample := rbSet[rand.Intn(len(rbSet))]
	sample.Description = " Description_" + RandomStr()
	bytes, _ := json.Marshal(sample)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	delete(payload, "users")
	delete(payload, "groups")
	delete(payload, "projects")
	payload["any_project"] = true
	delete(payload, "service_accounts")
	delete(payload, "valid_till")
	payload["ttl"] = 1000
	// payload["members"] = // fill with before created members
	return payload
}
