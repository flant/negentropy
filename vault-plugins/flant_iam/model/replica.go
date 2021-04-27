package model

import (
	"crypto/rsa"
	"encoding/json"

	"github.com/hashicorp/go-memdb"
)

const (
	ReplicaType = "replica" // also, memdb schema name

)

type Replica struct {
	Name      string         `json:"name"`
	TopicType string         `json:"type"`
	PublicKey *rsa.PublicKey `json:"replica_key"`
}

func (r Replica) Marshal(_ bool) ([]byte, error) {
	return json.Marshal(r)
}

func (r Replica) ObjType() string {
	return ReplicaType
}

func (r Replica) ObjId() string {
	return r.Name
}

func (r *Replica) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, r)
	return err
}

func ReplicaSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ReplicaType: {
				Name: ReplicaType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
					"type": {
						Name:   "type",
						Unique: false,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TopicType",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}
