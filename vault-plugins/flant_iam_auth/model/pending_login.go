package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"
)

const (
	PendingLoginType = "pending_login"
)

type PendingLogin struct {
	ID string `json:"id"`
}

func (pl PendingLogin) Marshal(_ bool) ([]byte, error) {
	return json.Marshal(pl)
}

func (pl PendingLogin) ObjType() string {
	return PendingLoginType
}

func (pl PendingLogin) ObjId() string {
	return pl.ID
}

func (pl *PendingLogin) Unmarshal(data []byte) error {
	return json.Unmarshal(data, pl)
}

func PendingLoginSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			PendingLoginType: {
				Name: PendingLoginType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "ID",
						},
					},
				},
			},
		},
	}
}
