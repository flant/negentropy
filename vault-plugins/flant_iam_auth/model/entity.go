package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"
)

type Entity struct {
	ID string `json:"id"`
}

const (
	EntityType = "entity"
)

func EntitySchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			EntityType: {
				Name: EntityType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
				},
			},
		},
	}
}

func (ent *Entity) ObjType() string {
	return EntityType
}

func (ent *Entity) ObjId() string {
	return ent.ID
}

func (ent Entity) Marshal(_ bool) ([]byte, error) {
	return json.Marshal(ent)
}

func (ent *Entity) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, ent)
	return err
}
