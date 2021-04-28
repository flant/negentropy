package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	GroupType = "group" // also, memdb schema name

)

type GroupObjectType string

func GroupSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			GroupType: {
				Name: GroupType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}

/*
identifier – уникален в рамках тенанта для каждого builtin_type_name
Пользователи
Группы
Сервисные аккаунты

uuid
tenant_uuid
identifier
full_identifier:

users
service_accounts
groups
resource_version
*/
type Group struct {
	UUID            string   `json:"uuid"` // ID
	TenantUUID      string   `json:"tenant_uuid"`
	Version         string   `json:"resource_version"`
	BuiltinType     string   `json:"-"`
	Identifier      string   `json:"identifier"`
	FullIdentifier  string   `json:"full_identifier"`
	Users           []string `json:"users"`
	Groups          []string `json:"groups"`
	ServiceAccounts []string `json:"service_accounts"`
}

func (u *Group) ObjType() string {
	return GroupType
}

func (u *Group) ObjId() string {
	return u.UUID
}

func (u *Group) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(u)
}

func (u *Group) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}

// generic: <identifier>@group.<tenant_identifier>
// builtin: <identifier>@<builtin_group_type>.group.<tenant_identifier>
func CalcGroupFullIdentifier(g *Group, tenant *Tenant) string {
	name := g.Identifier
	domain := "group." + tenant.Identifier
	if g.BuiltinType != "" {
		domain = g.BuiltinType + "." + domain
	}
	return name + "@" + domain
}
