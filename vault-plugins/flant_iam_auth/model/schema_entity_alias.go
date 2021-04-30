package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	EntityAliasType   = "entity_alias" // also, memdb schema name
	EntityAliasSource = "entity_alias_source"
	EntityName        = "entity_name"
)

func EntityAliasSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			EntityAliasType: {
				Name: EntityAliasType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					EntityAliasSource: {
						Name: EntityAliasSource,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{
									Field: "UserId",
								},

								&memdb.StringFieldIndex{
									Field: "SourceIdentifier",
								},
							},
						},
					},
					ByName: {
						Name: ByName,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
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

type EntityAlias struct {
	UUID             string `json:"uuid"`    // ID
	UserId           string `json:"user_id"` // our id
	Name             string `json:"name"`    // source name. by it vault look alias for user
	SourceIdentifier string `json:"source_identifier"`
}

func (p *EntityAlias) ObjType() string {
	return EntityAliasType
}

func (p *EntityAlias) ObjId() string {
	return p.UUID
}

func (p *EntityAlias) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(p)
}

func (p *EntityAlias) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, p)
	return err
}
