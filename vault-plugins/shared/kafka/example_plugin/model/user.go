package model

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/hashicorp/go-memdb"
)

type User = model.User

const UserType = model.UserType

func UserSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			UserType: {
				Name: UserType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					"version": {
						Name: "version",
						Indexer: &memdb.StringFieldIndex{
							Field: "Version",
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &memdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
				},
			},
		},
	}
}

type Marshaller interface {
	Marshal(bool) ([]byte, error)
	Unmarshal([]byte) error
}
