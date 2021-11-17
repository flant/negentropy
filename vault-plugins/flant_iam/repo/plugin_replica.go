package repo

import (
	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func ReplicaSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.ReplicaType: {
				Name: model.ReplicaType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Name",
						},
					},
					"type": {
						Name:   "type",
						Unique: false,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "TopicType",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}
