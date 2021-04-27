package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	TenantType      = "tenant" // also, memdb schema name
	TenantForeignPK = "tenant_uuid"
)

type Tenant struct {
	UUID       string `json:"uuid"` // ID
	Identifier string `json:"identifier"`
}

func (t *Tenant) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(t)
}

func (t *Tenant) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, t)
	return err
}

func TenantSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			TenantType: {
				Name: TenantType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					"identifier": {
						Name:   "identifier",
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field:     "Identifier",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}
