package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	RoleBindingType = "role_binding" // also, memdb schema name

)

type RoleBindingObjectType string

func RoleBindingSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			RoleBindingType: {
				Name: RoleBindingType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
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

type RoleBinding struct {
	UUID        string `json:"uuid"` // PK
	TenantUUID  string `json:"tenant_uuid"`
	Version     string `json:"resource_version"`
	BuiltinType string `json:"-"`

	ValidTill  int64 `json:"valid_till"`
	RequireMFA bool  `json:"require_mfa"`

	Users           []string `json:"users"`
	Groups          []string `json:"groups"`
	ServiceAccounts []string `json:"service_accounts"`

	Roles                    []BoundRole               `json:"-"`
	MaterializedRoles        []MaterializedRole        `json:"-"`
	MaterializedProjectRoles []MaterializedProjectRole `json:"-"`
}

func (u *RoleBinding) ObjType() string {
	return RoleBindingType
}

func (u *RoleBinding) ObjId() string {
	return u.UUID
}

func (u *RoleBinding) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(u)
}

func (u *RoleBinding) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}

type BoundRole struct {
	Name       string                 `json:"name"`
	Version    string                 `json:"resource_version"`
	AnyProject bool                   `json:"any_project"`
	Projects   []string               `json:"projects"`
	Options    map[string]interface{} `json:"options"`
}

type MaterializedRole struct {
	Name    string                 `json:"name"`
	Options map[string]interface{} `json:"options"`
}

type MaterializedProjectRole struct {
	Project string `json:"project"`
	Name    string `json:"name"`
}
