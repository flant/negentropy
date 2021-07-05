package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	ExtensionType = "extension" // also, memdb schema name
)

func ExtensionSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ExtensionType: {
				Name: ExtensionType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					"parent": {
						Name:   "parent",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "ParentType", Lowercase: true},
								&memdb.StringFieldIndex{Field: "ParentUUID", Lowercase: true},
							},
						},
					},
				},
			},
		},
	}
}

type Extension struct {
	UUID string `json:"uuid"` // PK

	// Origin is the source where the extension originates from
	Origin string `json:"origin"`

	// ParentType is the object type to which the extension belongs to, e.g. "User" or "ServiceAccount".
	ParentType string `json:"parent_type"`
	// ParentUUID is the id of an owner object
	ParentUUID string `json:"parent_uuid"`

	// Attributes is the data to pass to other systems transparently
	Attributes map[string]interface{} `json:"attributes"`
	// SensitiveAttributes is the data to pass to some other systems transparently
	SensitiveAttributes map[string]interface{} `json:"sensitive_attributes"`
}

func (t *Extension) ObjType() string {
	return ExtensionType
}

func (t *Extension) ObjId() string {
	return t.UUID
}

func (t *Extension) Marshal(_ bool) ([]byte, error) {
	// TODO exclude sensitive data
	return jsonutil.EncodeJSON(t)
}

func (t *Extension) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, t)
	return err
}
