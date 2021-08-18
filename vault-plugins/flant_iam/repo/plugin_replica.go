package repo

import (
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func ReplicaSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.ReplicaType: {
				Name: model.ReplicaType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
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
