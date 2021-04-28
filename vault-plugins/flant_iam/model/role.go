package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	RoleType = "role" // also, memdb schema name
)

func RoleSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			RoleType: {
				Name: RoleType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
					"type": {
						Name: "type",
						Indexer: &memdb.StringFieldIndex{
							Field: "Type",
						},
					},
				},
			},
		},
	}
}

type GroupScope string

const (
	GroupScopeTenant  GroupScope = "tenant"
	GroupScopeProject GroupScope = "project"
)

type Role struct {
	Name string     `json:"name"`
	Type GroupScope `json:"type"`

	Description   string `json:"description"`
	OptionsSchema string `json:"options_schema"`

	RequireOneOfFeatureFlags []string `json:"require_one_of_feature_flags"`
}

func (t *Role) ObjType() string {
	return RoleType
}

func (t *Role) ObjId() string {
	return t.Name
}

func (t *Role) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(t)
}

func (t *Role) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, t)
	return err
}
