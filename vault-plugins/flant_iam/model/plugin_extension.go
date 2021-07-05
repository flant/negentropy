package model

import (
	"crypto/rsa"
	"encoding/json"

	"github.com/hashicorp/go-memdb"
)

const (
	PluginExtensionType = "plugin_extension" // also, memdb schema name
)

type PluginExtension struct {
	Name          string         `json:"name"`
	OwnedTypes    []string       `json:"owned_types"`
	ExtendedTypes []string       `json:"extended_types"`
	AllowedRoles  []string       `json:"allowed_roles"`
	PublicKey     *rsa.PublicKey `json:"replica_key"`
}

func (r PluginExtension) Marshal(_ bool) ([]byte, error) {
	return json.Marshal(r)
}

func (r PluginExtension) ObjType() string {
	return PluginExtensionType
}

func (r PluginExtension) ObjId() string {
	return r.Name
}

func (r *PluginExtension) Unmarshal(data []byte) error {
	return json.Unmarshal(data, r)
}

func PluginExtensionSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			PluginExtensionType: {
				Name: PluginExtensionType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}
