package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	EntityType = "entity" // also, memdb schema name
)

func EntitySchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			EntityType: {
				Name: EntityType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByUserID: {
						Name: ByUserID,
						Indexer: &memdb.StringFieldIndex{
							Field: "UserId",
						},
					},
				},
			},
		},
	}
}

type Entity struct {
	UUID   string `json:"uuid"` // ID
	Name   string `json:"name"` // Identifier
	UserId string `json:"user_id"`
}

func (p *Entity) ObjType() string {
	return EntityType
}

func (p *Entity) ObjId() string {
	return p.UUID
}

func (p *Entity) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(p)
}

func (p *Entity) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, p)
	return err
}