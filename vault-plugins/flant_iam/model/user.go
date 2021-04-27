package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	UserType = "user" // also, memdb schema name

)

type User struct {
	UUID       string `json:"uuid"` // ID
	TenantUUID string `json:"tenant_uuid"`
}

func (t *User) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(t)
}

func (t *User) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, t)
	return err
}

func UserSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			UserType: {
				Name: UserType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name:   TenantForeignPK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}
